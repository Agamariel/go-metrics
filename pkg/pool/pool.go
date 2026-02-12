package pool

import "sync"

// Resetter определяет интерфейс для типов, которые могут сбрасывать своё состояние.
type Resetter interface {
	Reset()
}

// Pool представляет собой типобезопасную обёртку над sync.Pool
// для объектов, реализующих интерфейс Resetter.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт и возвращает новый Pool с заданной фабричной функцией.
// Параметр fn используется для создания новых экземпляров типа T,
// когда пул пуст.
func New[T Resetter](fn func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return fn()
			},
		},
	}
}

// Get извлекает объект из пула. Если пул пуст, создаёт новый объект
// с помощью фабричной функции, переданной в конструктор.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put сбрасывает состояние объекта и помещает его обратно в пул
// для последующего повторного использования.
func (p *Pool[T]) Put(x T) {
	x.Reset()
	p.pool.Put(x)
}
