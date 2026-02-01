# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Производительность и оптимизация памяти

### Бенчмарки

Проект включает бенчмарки для ключевых компонентов системы, измеряющие скорость выполнения и потребление памяти.

#### MemStorage (Repository)

```
BenchmarkMemStorage_UpdateMetric/Gauge-24         	19234776	        58.23 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemStorage_UpdateMetric/Counter-24       	21256808	        57.83 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemStorage_UpdateMetrics/Size_10-24      	 4704778	       256.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemStorage_UpdateMetrics/Size_100-24     	  442707	      2789 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemStorage_UpdateMetrics/Size_1000-24    	   44455	     26590 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemStorage_GetMetric/Gauge-24            	20395675	        57.73 ns/op	       8 B/op	       1 allocs/op
BenchmarkMemStorage_GetMetric/Counter-24          	20580189	        57.71 ns/op	       8 B/op	       1 allocs/op
BenchmarkMemStorage_GetAllMetrics/Size_10-24      	 1785986	       701.6 ns/op	     784 B/op	      11 allocs/op
BenchmarkMemStorage_GetAllMetrics/Size_100-24     	  222932	      5673 ns/op	    7328 B/op	     101 allocs/op
BenchmarkMemStorage_GetAllMetrics/Size_1000-24    	   21910	     57654 ns/op	   73537 B/op	    1001 allocs/op
```

#### MetricsService

```
BenchmarkMetricsService_UpdateMetricByPath/Gauge-24         	 5200378	       200.8 ns/op	      56 B/op	       2 allocs/op
BenchmarkMetricsService_UpdateMetricByPath/Counter-24       	 6682101	       176.3 ns/op	      56 B/op	       2 allocs/op
BenchmarkMetricsService_UpdateMetrics/Size_10-24            	 4734255	       252.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkMetricsService_UpdateMetrics/Size_100-24           	  469359	      2652 ns/op	       0 B/op	       0 allocs/op
BenchmarkMetricsService_UpdateMetrics/Size_1000-24          	   43527	     27013 ns/op	       0 B/op	       0 allocs/op
BenchmarkMetricsService_GetMetric/Gauge-24                  	17128860	        72.62 ns/op	       8 B/op	       1 allocs/op
BenchmarkMetricsService_GetMetric/Counter-24                	15695937	        73.24 ns/op	       8 B/op	       1 allocs/op
BenchmarkMetricsService_GetAllMetrics/Size_10-24            	 1701292	       712.9 ns/op	     784 B/op	      11 allocs/op
BenchmarkMetricsService_GetAllMetrics/Size_100-24           	  190772	      5420 ns/op	    7328 B/op	     101 allocs/op
BenchmarkMetricsService_GetAllMetrics/Size_1000-24          	   20768	     59033 ns/op	   73537 B/op	    1001 allocs/op
```

#### SHA256 Hash

```
BenchmarkCalculateSHA256/Size_100B-24         	 1087896	      1077 ns/op	  92.82 MB/s	     656 B/op	       9 allocs/op
BenchmarkCalculateSHA256/Size_1024B-24        	  501854	      2173 ns/op	 471.22 MB/s	     656 B/op	       9 allocs/op
BenchmarkCalculateSHA256/Size_10240B-24       	  100564	     11942 ns/op	 857.50 MB/s	     656 B/op	       9 allocs/op
BenchmarkCalculateSHA256/Size_102400B-24      	   10000	    111550 ns/op	 917.97 MB/s	     656 B/op	       9 allocs/op
BenchmarkVerifySHA256/Size_100B-24            	  931807	      1082 ns/op	  92.44 MB/s	     560 B/op	       8 allocs/op
BenchmarkVerifySHA256/Size_1024B-24           	  542536	      2107 ns/op	 486.00 MB/s	     560 B/op	       8 allocs/op
BenchmarkVerifySHA256/Size_10240B-24          	   96538	     12136 ns/op	 843.79 MB/s	     560 B/op	       8 allocs/op
BenchmarkVerifySHA256/Size_102400B-24         	   10000	    111566 ns/op	 917.84 MB/s	     560 B/op	       8 allocs/op
BenchmarkCalculateSHA256_NoKey-24             	444509803	         2.889 ns/op	       0 B/op	       0 allocs/op
```

#### Agent (JSON & Compression)

```
BenchmarkCompressJSON/Metrics_10-24         	    5390	    296511 ns/op	   1.49 MB/s	  814373 B/op	      21 allocs/op
BenchmarkCompressJSON/Metrics_100-24        	    3481	    363698 ns/op	  12.60 MB/s	  815778 B/op	      22 allocs/op
BenchmarkCompressJSON/Metrics_1000-24       	    1641	    678382 ns/op	  70.43 MB/s	  830112 B/op	      25 allocs/op
BenchmarkJSONMarshal/Metrics_10-24          	  469916	      2900 ns/op	     472 B/op	       2 allocs/op
BenchmarkJSONMarshal/Metrics_100-24         	   43168	     25956 ns/op	    4894 B/op	       2 allocs/op
BenchmarkJSONMarshal/Metrics_1000-24        	    4898	    293798 ns/op	   49305 B/op	       2 allocs/op
BenchmarkGzipCompression/BestSpeed-24       	    5779	    189194 ns/op	 1208982 B/op	      24 allocs/op
BenchmarkGzipCompression/Default-24         	    5943	    219945 ns/op	  815767 B/op	      22 allocs/op
BenchmarkGzipCompression/BestCompression-24 	    3908	    308171 ns/op	  815768 B/op	      22 allocs/op
```

### Профилирование памяти с pprof

#### Анализ и оптимизация

Проведен анализ использования памяти с помощью профилировщика pprof. Профили сохранены в директории `profiles/`.

##### Исходное состояние (base.pprof)

Анализ базового профиля выявил основной источник аллокаций:

```
Type: alloc_space
Showing nodes accounting for 53.95MB, 100% of 53.95MB total
      flat  flat%   sum%        cum   cum%
   49.93MB 92.56% 92.56%    49.93MB 92.56%  GetAllMetrics
    0.51MB  0.94% 98.15%     0.51MB  0.94%  UpdateMetric
```

Функция `GetAllMetrics` использовала **92.56%** всех аллокаций (49.93MB).

##### Проблемы и оптимизации

**Проблема:** В функции `GetAllMetrics` не выделялась память заранее для среза результатов, что приводило к множественным реаллокациям при каждом `append`.

**Решение:** Предварительное выделение памяти для среза с нужной capacity:

```go
// До оптимизации
var all []models.Metrics
for id, val := range m.gauges {
    all = append(all, ...)  // Множественные реаллокации
}

// После оптимизации
totalLen := len(m.gauges) + len(m.counters)
all := make([]models.Metrics, 0, totalLen)  // Одна аллокация
for id, val := range m.gauges {
    all = append(all, ...)  // Без реаллокаций
}
```

Аналогичная оптимизация применена в:
- `handler.ListMetricsHandler`
- `agent.SendAllMetrics`

##### Результаты оптимизации (result.pprof)

Сравнение профилей показало значительное улучшение:

```
pprof -alloc_space -top -diff_base=profiles/base.pprof profiles/result.pprof

Type: alloc_space
Showing nodes accounting for -36.15MB, 67.01% of 53.95MB total
      flat  flat%   sum%        cum   cum%
  -34.64MB 64.22% 64.22%   -34.64MB 64.22%  GetAllMetrics
   -0.51MB  0.94% 65.16%    -0.51MB  0.94%  UpdateMetric
```

**Уменьшение использования памяти:**
- GetAllMetrics: **-34.64MB** (уменьшение на 64.22%)
- UpdateMetric: **-0.51MB** (уменьшение на 0.94%)
- **Общее уменьшение: -36.15MB** (67.01% от исходного объема)

```
pprof -alloc_objects -top -diff_base=profiles/base.pprof profiles/result.pprof

Type: alloc_objects
Showing nodes accounting for -104613, 50.58% of 206833 total
      flat  flat%   sum%        cum   cum%
    -99152 47.94% 47.94%     -99152 47.94%  GetAllMetrics
```

**Уменьшение количества аллокаций:**
- GetAllMetrics: **-99,152 аллокаций** (уменьшение на 47.94%)
- **Общее уменьшение: -104,613 аллокаций** (50.58% от исходного количества)

### Запуск бенчмарков

```bash
# Repository
go test -bench=BenchmarkMemStorage -benchmem -run=^$ ./internal/repository

# Service
go test -bench=BenchmarkMetricsService -benchmem -run=^$ ./internal/service

# SHA256 Hash
go test -bench=Benchmark -benchmem -run=^$ ./pkg/sha256hash

# Agent
go test -bench=Benchmark -benchmem -run=^$ ./internal/agent
```

### Запуск профилирования

```bash
# Создание профиля
go run cmd/profiler/main.go profiles/output.pprof

# Анализ профиля
go tool pprof -top profiles/output.pprof
go tool pprof -alloc_space -top profiles/output.pprof

# Сравнение профилей
go tool pprof -alloc_space -top -diff_base=profiles/base.pprof profiles/result.pprof
```
