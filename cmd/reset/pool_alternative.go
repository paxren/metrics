package main

import "sync"

// ResetterPtr определяет интерфейс для указателей на типы, которые могут быть сброшены
type ResetterPtr[T any] interface {
	*T
	Reset()
}

// PoolPtr представляет собой пул указателей на объекты с методом Reset()
// Эта версия использует более явный подход для указателей
type PoolPtr[T any] struct {
	pool sync.Pool
}

// NewPoolPtr создаёт и возвращает указатель на структуру PoolPtr
func NewPoolPtr[T any](newFunc func() *T) *PoolPtr[T] {
	return &PoolPtr[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
	}
}

// Get возвращает указатель на объект из пула
func (p *PoolPtr[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put помещает указатель на объект в пул
func (p *PoolPtr[T]) Put(obj *T) {
	if obj != nil {
		// Приводим к интерфейсу Resetter для вызова метода Reset
		resetter := any(obj).(interface{ Reset() })
		resetter.Reset()
		p.pool.Put(obj)
	}
}
