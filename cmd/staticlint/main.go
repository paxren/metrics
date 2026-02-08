package main

import (
	"encoding/json"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"github.com/paxren/metrics/cmd/staticlint/osexit"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/structtag"
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

var ErrCheckAnalyzer = &analysis.Analyzer{
	Name: "errcheck",
	Doc:  "check for unchecked errors",
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

type Pass struct {
	// отобразим здесь только важные поля
	Fset         *token.FileSet // информация о позиции токенов
	Files        []*ast.File    // AST для каждого файла
	OtherFiles   []string       // имена файлов не на Go в пакете
	IgnoredFiles []string       // имена игнорируемых исходных файлов в пакете
	Pkg          *types.Package // информация о типах пакета
	TypesInfo    *types.Info    // информация о типах в AST
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

func isErrorType(t types.Type) bool {
	// проверяем, что t реализует интерфейс, при помощи которого определен тип error,
	// т.е. для типа t определен метод Error() string
	return types.Implements(t, errorType)
}

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

// isReturnError возвращает true, если среди возвращаемых значений есть ошибка.
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
type ConfigData struct {
	Staticcheck []string
}

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
		ErrCheckAnalyzer,
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
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
	multichecker.Main(
		//ErrCheckAnalyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		structtag.Analyzer,
		osexit.Analyzer,
	)
}
