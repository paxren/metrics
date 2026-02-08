package main

import "fmt"

// ExampleUsageAlternative демонстрирует использование альтернативной структуры PoolPtr
func ExampleUsageAlternative() {
	// Создаём пул для указателей на ExampleStruct
	pool := NewPoolPtr(func() *ExampleStruct {
		return &ExampleStruct{
			ID:     0,
			Name:   "",
			Active: false,
			Data:   make([]byte, 0, 1024), // Предварительно выделяем память
		}
	})

	// Получаем объект из пула
	obj1 := pool.Get()
	fmt.Printf("Полученный объект: %+v\n", obj1)

	// Модифицируем объект
	obj1.ID = 1
	obj1.Name = "Test"
	obj1.Active = true
	obj1.Data = append(obj1.Data, []byte("some data")...)
	fmt.Printf("Модифицированный объект: %+v\n", obj1)

	// Возвращаем объект в пул (он будет автоматически сброшен)
	pool.Put(obj1)

	// Получаем другой объект из пула (может быть тот же самый, но сброшенный)
	obj2 := pool.Get()
	fmt.Printf("Объект после возврата в пул: %+v\n", obj2)

	// Пул можно использовать для снижения аллокаций в высоконагруженных системах
	// Например, в обработчиках запросов:
	processRequestAlternative(pool)
}

// processRequestAlternative пример функции, использующей альтернативный пул для обработки запросов
func processRequestAlternative(pool *PoolPtr[ExampleStruct]) {
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
