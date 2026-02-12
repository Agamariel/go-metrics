package pool

import (
	"testing"
)

// testStruct — тестовая структура с методом Reset()
type testStruct struct {
	Counter int
	Items   []string
	Active  bool
}

func (t *testStruct) Reset() {
	t.Counter = 0
	t.Items = t.Items[:0]
	t.Active = false
}

// TestNew проверяет создание нового пула
func TestNew(t *testing.T) {
	p := New(func() *testStruct {
		return &testStruct{}
	})

	if p == nil {
		t.Fatal("New() returned nil")
	}
}

// TestGet проверяет получение объекта из пула
func TestGet(t *testing.T) {
	factoryCalled := false
	p := New(func() *testStruct {
		factoryCalled = true
		return &testStruct{Counter: 42}
	})

	obj := p.Get()

	if !factoryCalled {
		t.Error("Factory function was not called")
	}

	if obj == nil {
		t.Fatal("Get() returned nil")
	}

	if obj.Counter != 42 {
		t.Errorf("Expected Counter to be 42, got %d", obj.Counter)
	}
}

// TestPutResetsState проверяет, что Put вызывает Reset()
func TestPutResetsState(t *testing.T) {
	p := New(func() *testStruct {
		return &testStruct{}
	})

	obj := p.Get()
	obj.Counter = 100
	obj.Items = append(obj.Items, "test1", "test2")
	obj.Active = true

	// Возвращаем объект в пул
	p.Put(obj)

	// Проверяем, что Reset() был вызван
	if obj.Counter != 0 {
		t.Errorf("Expected Counter to be 0 after Put, got %d", obj.Counter)
	}
	if len(obj.Items) != 0 {
		t.Errorf("Expected Items to be empty after Put, got %v", obj.Items)
	}
	if obj.Active {
		t.Error("Expected Active to be false after Put")
	}
}

// TestPoolReuse проверяет повторное использование объектов из пула
func TestPoolReuse(t *testing.T) {
	callCount := 0
	p := New(func() *testStruct {
		callCount++
		return &testStruct{}
	})

	// Получаем первый объект (фабрика должна вызваться)
	obj1 := p.Get()
	if callCount != 1 {
		t.Errorf("Expected factory to be called once, was called %d times", callCount)
	}

	obj1.Counter = 50
	obj1.Items = append(obj1.Items, "data")

	// Возвращаем объект в пул
	p.Put(obj1)

	// Получаем объект снова (фабрика НЕ должна вызваться, т.к. объект из пула)
	obj2 := p.Get()
	if callCount != 1 {
		t.Errorf("Expected factory to be called only once, was called %d times", callCount)
	}

	// obj2 должен быть тем же объектом, что и obj1, но сброшенным
	if obj2.Counter != 0 {
		t.Errorf("Expected Counter to be 0, got %d", obj2.Counter)
	}
	if len(obj2.Items) != 0 {
		t.Errorf("Expected Items to be empty, got %v", obj2.Items)
	}
}

// TestMultipleObjects проверяет работу с несколькими объектами
func TestMultipleObjects(t *testing.T) {
	p := New(func() *testStruct {
		return &testStruct{}
	})

	// Получаем несколько объектов
	obj1 := p.Get()
	obj2 := p.Get()
	obj3 := p.Get()

	obj1.Counter = 1
	obj2.Counter = 2
	obj3.Counter = 3

	// Возвращаем их в пул
	p.Put(obj1)
	p.Put(obj2)
	p.Put(obj3)

	// Все объекты должны быть сброшены
	if obj1.Counter != 0 || obj2.Counter != 0 || obj3.Counter != 0 {
		t.Error("Not all objects were reset properly")
	}
}

// TestSliceCapacityPreserved проверяет, что capacity слайса сохраняется после Reset
func TestSliceCapacityPreserved(t *testing.T) {
	p := New(func() *testStruct {
		return &testStruct{
			Items: make([]string, 0, 10), // capacity = 10
		}
	})

	obj := p.Get()
	initialCap := cap(obj.Items)

	// Добавляем элементы
	for i := 0; i < 5; i++ {
		obj.Items = append(obj.Items, "item")
	}

	// Возвращаем в пул
	p.Put(obj)

	// Проверяем, что length = 0, но capacity сохранилась
	if len(obj.Items) != 0 {
		t.Errorf("Expected length to be 0, got %d", len(obj.Items))
	}
	if cap(obj.Items) != initialCap {
		t.Errorf("Expected capacity to be %d, got %d", initialCap, cap(obj.Items))
	}
}

// TestConcurrentAccess проверяет базовую потокобезопасность
func TestConcurrentAccess(t *testing.T) {
	p := New(func() *testStruct {
		return &testStruct{}
	})

	done := make(chan bool)

	// Запускаем несколько горутин, которые одновременно работают с пулом
	for i := 0; i < 100; i++ {
		go func(id int) {
			obj := p.Get()
			obj.Counter = id
			obj.Items = append(obj.Items, "test")
			p.Put(obj)
			done <- true
		}(i)
	}

	// Ждём завершения всех горутин
	for i := 0; i < 100; i++ {
		<-done
	}

	// Если тест дошёл сюда без паники или race condition, значит всё ок
}

// BenchmarkPoolGetPut измеряет производительность Get/Put
func BenchmarkPoolGetPut(b *testing.B) {
	p := New(func() *testStruct {
		return &testStruct{
			Items: make([]string, 0, 10),
		}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := p.Get()
		obj.Counter = i
		obj.Items = append(obj.Items, "benchmark")
		p.Put(obj)
	}
}

// BenchmarkPoolParallel измеряет производительность при параллельном доступе
func BenchmarkPoolParallel(b *testing.B) {
	p := New(func() *testStruct {
		return &testStruct{
			Items: make([]string, 0, 10),
		}
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := p.Get()
			obj.Counter++
			obj.Items = append(obj.Items, "parallel")
			p.Put(obj)
		}
	})
}
