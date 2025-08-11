// cmd/staticlint/exitcheck.go
package main

import (
	"go/ast"
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

func run(pass *analysis.Pass) (interface{}, error) { //nolint:gocognit
	for _, file := range pass.Files {
		// Проверяем только файлы пакета main
		if file.Name.Name != "main" {
			continue
		}

		pos := pass.Fset.Position(file.Pos())
		filename := pos.Filename

		// Пропускаем файлы из кэша сборки
		if strings.Contains(filename, "go-build") {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, okF := n.(*ast.FuncDecl)
			if !okF || funcDecl.Name.Name != "main" {
				return true
			}

			ast.Inspect(funcDecl, func(n ast.Node) bool {
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				ident, ok := selExpr.X.(*ast.Ident)
				if !ok || ident.Name != "os" || selExpr.Sel.Name != "Exit" {
					return true
				}

				position := pass.Fset.Position(callExpr.Pos())
				pass.Reportf(
					callExpr.Pos(),
					"%s:%d: direct call os.Exit is not allowed in main function",
					filepath.Base(position.Filename),
					position.Line,
				)
				return true
			})
			return true
		})
	}

	return nil, nil //nolint:nilnil
}
