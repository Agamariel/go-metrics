// Package osexit содержит анализатор, запрещающий использовать прямой вызов os.Exit
// в функции main пакета main.
package osexit

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer запрещает использование прямого вызова os.Exit в функции main пакета main.
//
// Прямой вызов os.Exit в main функции затрудняет тестирование и корректное завершение работы
// программы (например, отложенные вызовы defer не выполняются). Рекомендуется выносить логику
// в отдельную функцию (например, run()), которая возвращает ошибку, а main обрабатывает её
// через log.Fatal или fmt.Fprintf + os.Exit.
var Analyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Проверяем только пакет main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	// Обходим все файлы в пакете
	for _, file := range pass.Files {
		// Ищем функцию main
		ast.Inspect(file, func(node ast.Node) bool {
			// Ищем объявление функции
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// Проверяем, что это функция main (без receiver)
			if funcDecl.Name.Name != "main" || funcDecl.Recv != nil {
				return true
			}

			// Обходим тело функции main
			if funcDecl.Body != nil {
				ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
					// Ищем вызовы функций
					callExpr, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					// Проверяем, что это вызов os.Exit
					if isOsExitCall(callExpr) {
						// Проверяем, что вызов находится не во внешнем файле
						callFilename := pass.Fset.Position(callExpr.Pos()).Filename
						if !isExternalFile(callFilename) {
							pass.Reportf(callExpr.Pos(), "прямой вызов os.Exit в функции main запрещен")
						}
					}

					return true
				})
			}

			return true
		})
	}

	return nil, nil
}

// isExternalFile проверяет, является ли файл внешним (из кэша или зависимостей)
func isExternalFile(filename string) bool {
	filename = filepath.ToSlash(filename)
	
	// Пропускаем файлы из build cache
	if strings.Contains(filename, "/go-build/") || strings.Contains(filename, "\\go-build\\") {
		return true
	}
	
	// Пропускаем файлы из модулей (go/pkg/mod)
	if strings.Contains(filename, "/pkg/mod/") || strings.Contains(filename, "\\pkg\\mod\\") {
		return true
	}
	
	// Пропускаем файлы из стандартной библиотеки Go
	if strings.Contains(filename, "/Go/src/") || strings.Contains(filename, "\\Go\\src\\") {
		return true
	}
	
	return false
}

// isOsExitCall проверяет, является ли вызов функцией os.Exit
func isOsExitCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Проверяем, что X - это идентификатор "os"
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Проверяем имя пакета и функции
	return ident.Name == "os" && selector.Sel.Name == "Exit"
}
