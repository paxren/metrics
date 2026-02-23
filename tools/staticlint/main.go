package main

import (
	"encoding/json"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/paxren/metrics/tools/staticlint/osexit"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
)

// func main() {
// 	multichecker.Main(
// 		ErrCheckAnalyzer,
// 		printf.Analyzer,
// 		shadow.Analyzer,
// 		shift.Analyzer,
// 		structtag.Analyzer,
// 	)
// }

// ErrCheckAnalyzer проверяет наличие необработанных ошибок в коде.
// Анализатор находит следующие ситуации:
//  1. Вызовы функций, возвращающих ошибку, без обработки ошибки
//  2. Присваивание с использованием '_' для игнорирования ошибки
//  3. Множественное присваивание с игнорированием ошибки
//
// Примеры обнаруживаемых проблем:
//
//	func() error { return nil }()  // ошибка не обработана
//	_, _ := os.Open("file.txt")    // ошибка проигнорирована
//	a, _ := someFunc()             // ошибка проигнорирована
var ErrCheckAnalyzer = &analysis.Analyzer{
	Name: "errcheck",
	Doc:  "проверка необработанных ошибок",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	expr := func(x *ast.ExprStmt) {
		// проверяем, что выражение представляет собой вызов функции,
		// у которой возвращаемая ошибка никак не обрабатывается
		if call, ok := x.X.(*ast.CallExpr); ok {
			if isReturnError(pass, call) {
				pass.Reportf(x.Pos(), "expression returns unchecked error")
			}
		}
	}
	tuplefunc := func(x *ast.AssignStmt) {
		// рассматриваем присваивание, при котором
		// вместо получения ошибок используется '_'
		// a, b, _ := tuplefunc()
		// проверяем, что это вызов функции
		if call, ok := x.Rhs[0].(*ast.CallExpr); ok {
			results := resultErrors(pass, call)
			for i := 0; i < len(x.Lhs); i++ {
				// перебираем все идентификаторы слева от присваивания
				if id, ok := x.Lhs[i].(*ast.Ident); ok && id.Name == "_" && results[i] {
					pass.Reportf(id.NamePos, "assignment with unchecked error")
				}
			}
		}
	}
	errfunc := func(x *ast.AssignStmt) {
		// множественное присваивание: a, _ := b, myfunc()
		// ищем ситуацию, когда функция справа возвращает ошибку,
		// а соответствующий идентификатор слева равен '_'
		for i := 0; i < len(x.Lhs); i++ {
			if id, ok := x.Lhs[i].(*ast.Ident); ok {
				// вызов функции справа
				if call, ok := x.Rhs[i].(*ast.CallExpr); ok {
					if id.Name == "_" && isReturnError(pass, call) {
						pass.Reportf(id.NamePos, "assignment with unchecked error")
					}
				}
			}
		}
	}
	for _, file := range pass.Files {
		// функцией ast.Inspect проходим по всем узлам AST
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.ExprStmt: // выражение
				expr(x)
			case *ast.AssignStmt: // оператор присваивания
				// справа одно выражение x,y := myfunc()
				if len(x.Rhs) == 1 {
					tuplefunc(x)
				} else {
					// справа несколько выражений x,y := z,myfunc()
					errfunc(x)
				}
			}
			return true
		})
	}
	return nil, nil
}

// Pass содержит информацию о проходе анализатора.
// Эта структура является упрощенной версией analysis.Pass и содержит
// только основные поля, необходимые для работы анализатора.
type Pass struct {
	// Fset содержит информацию о позициях токенов в исходных файлах
	Fset *token.FileSet
	// Files содержит AST для каждого файла в пакете
	Files []*ast.File
	// OtherFiles содержит имена файлов не на Go в пакете
	OtherFiles []string
	// IgnoredFiles содержит имена игнорируемых исходных файлов в пакете
	IgnoredFiles []string
	// Pkg содержит информацию о типах пакета
	Pkg *types.Package
	// TypesInfo содержит информацию о типах в AST
	TypesInfo *types.Info
}

var errorType = types.
	// ищем тип error в области вилимости Universe, в котором находятся
	// все предварительно объявленные объекты Go
	Universe.Lookup("error").
	// получаем объект, представляющий тип error
	Type().
	// получаем тип, при помощи которого определен тип error (см. https://go.dev/ref/spec#Underlying_types);
	// мы знаем, что error определен как интерфейс, приведем полученный объект к этому типу
	Underlying().(*types.Interface)

// isErrorType проверяет, что тип t реализует интерфейс error.
// Функция возвращает true, если для типа t определен метод Error() string,
// что означает соответствие интерфейсу error.
func isErrorType(t types.Type) bool {
	// проверяем, что t реализует интерфейс, при помощи которого определен тип error,
	// т.е. для типа t определен метод Error() string
	return types.Implements(t, errorType)
}

// resultErrors определяет, какие из возвращаемых значений функции являются ошибками.
// Функция анализирует тип возвращаемого значения вызова функции и возвращает
// срез булевых значений, где true означает, что соответствующее возвращаемое
// значение является ошибкой.
func resultErrors(pass *analysis.Pass, call *ast.CallExpr) []bool {
	switch t := pass.TypesInfo.Types[call].Type.(type) {
	case *types.Named: // возвращается значение
		return []bool{isErrorType(t)}
	case *types.Pointer: // возвращается указатель
		return []bool{isErrorType(t)}
	case *types.Tuple: // возвращается несколько значений
		s := make([]bool, t.Len())
		for i := 0; i < t.Len(); i++ {
			switch mt := t.At(i).Type().(type) {
			case *types.Named:
				s[i] = isErrorType(mt)
			case *types.Pointer:
				s[i] = isErrorType(mt)
			}
		}
		return s
	}
	return []bool{false}
}

// isReturnError возвращает true, если среди возвращаемых значений вызова функции есть ошибка.
// Функция использует resultErrors для анализа типов возвращаемых значений и проверяет,
// что хотя бы одно из них является ошибкой.
func isReturnError(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, isError := range resultErrors(pass, call) {
		if isError {
			return true
		}
	}
	return false
}

// //////////////////
// Config — имя файла конфигурации.
const Config = `config.json`

// ConfigData описывает структуру файла конфигурации.
// Поле Staticcheck содержит список анализаторов из пакета staticcheck,
// которые должны быть включены в анализ.
type ConfigData struct {
	// Staticcheck содержит список имен анализаторов staticcheck для включения
	Staticcheck []string
}

// main является точкой входа в программу staticlint.
// Функция выполняет следующие шаги:
//  1. Определяет путь к исполняемому файлу
//  2. Загружает конфигурацию из файла config.json
//  3. Создает список анализаторов, включая:
//     - Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
//     - Публичные анализаторы (например, ineffassign)
//     - Пользовательские анализаторы (например, osexit)
//  4. Добавляет анализаторы из staticcheck в соответствии с конфигурацией
//  5. Запускает multichecker.Main для выполнения анализа
//
// multichecker.Main - это функция из пакета golang.org/x/tools/go/analysis/multichecker,
// которая запускает множество анализаторов над указанными пакетами.
// Она обрабатывает флаги командной строки, парсит пакеты и последовательно
// применяет каждый анализатор к коду.
func main() {
	appfile, err := os.Executable()
	if err != nil {
		panic(err)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(appfile), Config))
	if err != nil {
		panic(err)
	}
	var cfg ConfigData
	if err = json.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}
	mychecks := []*analysis.Analyzer{
		//ErrCheckAnalyzer,
		// Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		slog.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		// Публичные анализаторы
		ineffassign.Analyzer,
		// Пользовательские анализаторы
		osexit.Analyzer,
	}
	checks := make(map[string]bool)
	for _, v := range cfg.Staticcheck {
		checks[v] = true
	}
	// добавляем анализаторы из staticcheck, которые указаны в файле конфигурации
	for _, v := range staticcheck.Analyzers {
		if checks[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)
		}
	}
	multichecker.Main(mychecks...)
}
