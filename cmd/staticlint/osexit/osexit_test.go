package osexit_test

import (
	"testing"

	"github.com/Agamariel/go-metrics/cmd/staticlint/osexit"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestOsExitAnalyzer(t *testing.T) {
	// Создаём тестовую директорию с тестовыми пакетами
	testdata := analysistest.TestData()
	
	// Запускаем анализатор на тестовых данных
	analysistest.Run(t, testdata, osexit.Analyzer, "main")
}
