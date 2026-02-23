package main

import (
	"fmt"
	"testing"
)

// TestPoolExample демонстрирует работу пула в тесте
func TestPoolExample(t *testing.T) {
	// Создаём пул для указателей на ExampleStruct
	pool := New(func() *ExampleStruct {
		return &ExampleStruct{
			ID:     0,
			Name:   "",
			Active: false,
			Data:   make([]byte, 0, 1024),
		}
	})

	// Получаем объект из пула
	obj1 := pool.Get()
	if obj1.ID != 0 || obj1.Name != "" || obj1.Active != false || len(obj1.Data) != 0 {
		t.Errorf("Initial object state is incorrect: %+v", obj1)
	}

	// Модифицируем объект
	obj1.ID = 1
	obj1.Name = "Test"
	obj1.Active = true
	obj1.Data = append(obj1.Data, []byte("some data")...)

	// Возвращаем объект в пул
	pool.Put(obj1)

	// Получаем другой объект из пула (должен быть сброшен)
	obj2 := pool.Get()
	if obj2.ID != 0 || obj2.Name != "" || obj2.Active != false || len(obj2.Data) != 0 {
		t.Errorf("Object after reset is incorrect: %+v", obj2)
	}

	// Проверяем, что это тот же самый объект (оптимизация пула)
	if obj1 != obj2 {
		t.Log("Pool created a new object (expected behavior)")
	} else {
		t.Log("Pool reused the same object (expected behavior)")
	}
}

// BenchmarkPoolWithoutPool измеряет производительность без пула
func BenchmarkPoolWithoutPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		obj := &ExampleStruct{
			ID:     0,
			Name:   "",
			Active: false,
			Data:   make([]byte, 0, 1024),
		}

		obj.ID = i
		obj.Name = "benchmark"
		obj.Active = true
		obj.Data = append(obj.Data, []byte("benchmark data")...)

		// Имитируем сброс
		obj.Reset()
	}
}

// BenchmarkPoolWithPool измеряет производительность с пулом
func BenchmarkPoolWithPool(b *testing.B) {
	pool := New(func() *ExampleStruct {
		return &ExampleStruct{
			ID:     0,
			Name:   "",
			Active: false,
			Data:   make([]byte, 0, 1024),
		}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := pool.Get()

		obj.ID = i
		obj.Name = "benchmark"
		obj.Active = true
		obj.Data = append(obj.Data, []byte("benchmark data")...)

		pool.Put(obj)
	}
}

// ExamplePool демонстрирует использование пула
func ExamplePool() {
	// Создаём пул
	pool := New(func() *ExampleStruct {
		return &ExampleStruct{
			ID:     0,
			Name:   "",
			Active: false,
			Data:   make([]byte, 0, 1024),
		}
	})

	// Получаем объект
	obj := pool.Get()
	fmt.Printf("Полученный объект: %+v\n", obj)

	// Используем объект
	obj.ID = 42
	obj.Name = "Example"
	obj.Active = true

	// Возвращаем в пул
	pool.Put(obj)

	// Получаем снова (объект будет сброшен)
	obj = pool.Get()
	fmt.Printf("Сброшенный объект: %+v\n", obj)

	// Output:
	// Полученный объект: &{ID:0 Name: Active:false Data:[]}
	// Сброшенный объект: &{ID:0 Name: Active:false Data:[]}
}
