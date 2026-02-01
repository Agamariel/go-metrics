# Профили памяти (pprof)

Эта директория содержит профили памяти, созданные с помощью Go pprof для анализа и оптимизации использования памяти в проекте.

## Профили

- `base.pprof` - базовый профиль до оптимизаций
- `result.pprof` - профиль после оптимизаций

## Использование

### Создание профиля

```bash
go run cmd/profiler/main.go profiles/my-profile.pprof
```

### Анализ профиля

#### Просмотр топ потребителей памяти

```bash
# Текущее использование памяти (inuse_space)
go tool pprof -top profiles/base.pprof

# Все аллокации (alloc_space)
go tool pprof -alloc_space -top profiles/base.pprof

# Количество аллокаций (alloc_objects)
go tool pprof -alloc_objects -top profiles/base.pprof
```

#### Детальный анализ функции

```bash
go tool pprof -list=GetAllMetrics profiles/base.pprof
```

#### Интерактивный режим

```bash
go tool pprof profiles/base.pprof

# В интерактивном режиме доступны команды:
# top - показать топ функций
# list <функция> - показать исходный код функции с аннотациями
# web - открыть граф в браузере (требует graphviz)
# peek - показать вызовы функций
# help - справка по командам
```

#### Веб-интерфейс

```bash
go tool pprof -http=:8080 profiles/base.pprof
```

Откроется веб-интерфейс на http://localhost:8080 с графами, flame graphs и другими визуализациями.

### Сравнение профилей

```bash
# Сравнение по использованию памяти
go tool pprof -alloc_space -top -diff_base=profiles/base.pprof profiles/result.pprof

# Сравнение по количеству аллокаций
go tool pprof -alloc_objects -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Отрицательные значения в выводе означают уменьшение использования памяти или количества аллокаций.

## Результаты оптимизации

После применения оптимизаций (предварительное выделение памяти для срезов):

**Уменьшение использования памяти:** -36.15MB (67.01%)
**Уменьшение количества аллокаций:** -104,613 (50.58%)

Основной источник улучшений - функция `GetAllMetrics`, где было устранено 99,152 аллокаций.

## Рекомендации по профилированию

1. **Всегда делайте базовый профиль** перед оптимизациями для сравнения
2. **Сосредоточьтесь на горячих точках** - оптимизируйте функции с наибольшим потреблением
3. **Измеряйте результаты** - проверяйте эффект оптимизаций через diff профилей
4. **Используйте alloc_space** для оценки общего объема аллокаций
5. **Используйте alloc_objects** для оценки количества аллокаций
6. **Используйте inuse_space** для оценки текущей занятой памяти

## Дополнительные ресурсы

- [Официальная документация pprof](https://pkg.go.dev/runtime/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Практическое руководство по Go pprof](https://github.com/google/pprof/blob/main/doc/README.md)
