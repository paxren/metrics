package osexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "check for direct os.Exit calls in main function of main package",
	Run:  run,
}

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

// isOsExitCall проверяет, является ли вызов вызовом os.Exit
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
