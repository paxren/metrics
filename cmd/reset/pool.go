package main

import "sync"

// Resetter определяет интерфейс для типов, которые могут быть сброшены
type Resetter interface {
	Reset()
}

// Pool представляет собой пул указателей на объекты с методом Reset()
// Generic-параметр T ограничен типами, реализующими интерфейс Resetter
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт и возвращает указатель на структуру Pool
// Параметр newFunc - функция для создания нового объекта, когда пул пуст
func New[T Resetter](newFunc func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
	}
}

// Get возвращает объект из пула
// Если пул пуст, создаётся новый объект с помощью функции, переданной в New
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put помещает объект в пул
// Перед помещением объект сбрасывается в начальное состояние с помощью метода Reset()
func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
