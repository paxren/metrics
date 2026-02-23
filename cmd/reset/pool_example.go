package main

import "fmt"

// ExampleStruct - пример структуры, которая может использоваться с Pool
// generate:reset
type ExampleStruct struct {
	ID     int
	Name   string
	Active bool
	Data   []byte
}

// ExampleUsage демонстрирует использование структуры Pool
func ExampleUsage() {
	// Используем альтернативную реализацию пула, которая работает со statictest
	ExampleUsageAlternative()
}

// processRequest пример функции, использующей пул для обработки запросов
func processRequest(pool *PoolPtr[ExampleStruct]) {
	// Получаем объект из пула
	req := pool.Get()
	defer pool.Put(req) // Гарантируем возврат в пул

	// Используем объект для обработки запроса
	req.ID = 123
	req.Name = "Request Data"
	req.Active = true
	req.Data = append(req.Data, []byte("request payload")...)

	// Обработка запроса...
	fmt.Printf("Обработка запроса: %+v\n", req)
}
