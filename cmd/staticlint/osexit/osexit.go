// Package osexit реализует анализатор для проверки прямых вызовов os.Exit
// в функции main пакета main.
//
// Анализатор находит прямые вызовы os.Exit в функции main, что считается
// плохой практикой, так как это препятствует корректной очистке ресурсов
// и обработке ошибок в библиотечном коде.
package osexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer проверяет наличие прямых вызовов os.Exit в функции main пакета main.
// Анализатор находит следующие ситуации:
//  1. Прямые вызовы os.Exit в функции main
//  2. Вызовы os.Exit в любом месте пакета main (только в функции main)
//
// Пример обнаруживаемой проблемы:
//
//	func main() {
//	    os.Exit(1)  // прямой вызов os.Exit
//	}
//
// Рекомендуется использовать return вместо os.Exit в функции main.
var Analyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "проверка прямых вызовов os.Exit в функции main пакета main",
	Run:  run,
}

// run выполняет анализ пакета на наличие прямых вызовов os.Exit в функции main.
// Функция проверяет, что текущий пакет является пакетом main, затем ищет
// функцию main и проверяет наличие вызовов os.Exit в ее теле.
func run(pass *analysis.Pass) (interface{}, error) {
	// Проверяем, что мы находимся в пакете main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Проходим по всем узлам AST
		ast.Inspect(file, func(node ast.Node) bool {
			// Ищем объявления функций
			fn, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// Проверяем, что это функция main
			if fn.Name.Name != "main" {
				return true
			}

			// Проходим по телу функции main и ищем вызовы os.Exit
			ast.Inspect(fn, func(node ast.Node) bool {
				callExpr, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем, что это вызов os.Exit
				if isOsExitCall(pass, callExpr) {
					pass.Reportf(callExpr.Pos(), "direct os.Exit call in main function is not allowed")
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}

// isOsExitCall проверяет, является ли вызов вызовом os.Exit.
// Функция анализирует селекторное выражение и проверяет, что
// вызывается метод Exit из пакета os.
func isOsExitCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Проверяем для селекторных выражений (os.Exit)
	if selExpr, ok := call.Fun.(*ast.SelectorExpr); ok {
		if selExpr.Sel.Name == "Exit" {
			// Проверяем, что это пакет os
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				if ident.Name == "os" {
					return true
				}
			}
		}
	}

	return false
}
