package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0) // want "direct os.Exit call in main function is not allowed"

	if len(os.Args) > 1 {
		os.Exit(1) // want "direct os.Exit call in main function is not allowed"
	}

	// Проверяем, что анализатор не срабатывает на другие функции
	otherFunc()
}

func otherFunc() {
	// Здесь os.Exit должен быть разрешен, так как это не функция main
	os.Exit(2)
}

func indirectExit() {
	// Проверяем, что анализатор не срабатывает на косвенные вызовы
	exit := os.Exit
	exit(3)
}
