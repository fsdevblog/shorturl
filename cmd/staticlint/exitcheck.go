// cmd/staticlint/exitcheck.go
package main

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// NoDirectOsExit определяет анализатор, который проверяет прямые вызовы os.Exit
// в функции main пакета main.
// nolint:gochecknoglobals
var NoDirectOsExit = &analysis.Analyzer{
	Name: "nodirectosexit",
	Doc:  "check for direct os.Exit calls in main function",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		if !isMainPackageFile(file, pass.Fset) {
			continue
		}

		inspectMainFunction(file, pass)
	}

	return nil, nil //nolint:nilnil
}

// inspectMainFunction проверяет наличие прямых вызовов os.Exit в функции main.
func inspectMainFunction(file *ast.File, pass *analysis.Pass) {
	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "main" {
			return true
		}

		checkForOsExit(funcDecl, pass)
		return true
	})
}

// checkForOsExit проверяет наличие вызовов os.Exit в заданной функции.
func checkForOsExit(funcDecl *ast.FuncDecl, pass *analysis.Pass) {
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if isOsExitCall(callExpr) {
				reportOsExitUsage(callExpr, pass)
			}
		}
		return true
	})
}

// isOsExitCall проверяет, является ли выражение вызовом os.Exit.
func isOsExitCall(callExpr *ast.CallExpr) bool {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selExpr.X.(*ast.Ident)
	return ok && ident.Name == "os" && selExpr.Sel.Name == "Exit"
}

// reportOsExitUsage сообщает о найденном использовании os.Exit.
func reportOsExitUsage(callExpr *ast.CallExpr, pass *analysis.Pass) {
	position := pass.Fset.Position(callExpr.Pos())
	pass.Reportf(
		callExpr.Pos(),
		"%s:%d: direct call os.Exit is not allowed in main function",
		filepath.Base(position.Filename),
		position.Line,
	)
}

// isMainPackageFile проверяет, является ли файл частью пакета main и не находится в кэше сборки.
func isMainPackageFile(file *ast.File, fset *token.FileSet) bool {
	if file.Name.Name != "main" {
		return false
	}

	pos := fset.Position(file.Pos())
	return !strings.Contains(pos.Filename, "go-build")
}
