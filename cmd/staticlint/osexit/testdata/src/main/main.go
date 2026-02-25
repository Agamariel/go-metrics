package main

import (
	"fmt"
	"os"
)

func main() {
	// Прямой вызов os.Exit - должен быть обнаружен
	if true {
		os.Exit(1) // want "прямой вызов os.Exit в функции main запрещен"
	}

	// Еще один вызов
	fmt.Println("test")
	os.Exit(0) // want "прямой вызов os.Exit в функции main запрещен"
}

// Вспомогательная функция - вызов os.Exit здесь допустим
func helper() {
	os.Exit(1) // это не main функция, ошибки не должно быть
}
