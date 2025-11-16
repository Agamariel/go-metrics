package repository

import (
	"encoding/json"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	"github.com/Agamariel/go-metrics/internal/models"
	"go.uber.org/zap"
)

// ticker определяет минимальный интерфейс для таймера
type ticker interface {
	C() <-chan time.Time
	Stop()
}

// timeTicker — простая обёртка над time.Ticker
type timeTicker struct {
	*time.Ticker
}

func (t *timeTicker) C() <-chan time.Time {
	return t.Ticker.C
}

// FileStorage — обёртка над MemStorage с персистентностью в файл
type FileStorage struct {
	mem           *MemStorage
	filePath      string
	storeInterval int  // в секундах, 0 = синхронное сохранение
	syncWrite     bool // true если storeInterval == 0
	logger        logger.Logger
	mu            sync.RWMutex
	stopChan      chan struct{}
	ticker        ticker
	saveCount     atomic.Int64 // счетчик операций сохранения для отладки
}

// FileStorageConfig — конфигурация для FileStorage
type FileStorageConfig struct {
	FilePath      string
	StoreInterval int
	Restore       bool
	Logger        logger.Logger
	Ticker        ticker
}

// NewFileStorage создаёт новый FileStorage
func NewFileStorage(config FileStorageConfig) (*FileStorage, error) {
	log := config.Logger
	if log == nil {
		log = logger.Nop()
	}

	fs := &FileStorage{
		mem:           NewMemStorage(),
		filePath:      config.FilePath,
		storeInterval: config.StoreInterval,
		syncWrite:     config.StoreInterval == 0,
		logger:        log,
		stopChan:      make(chan struct{}),
		ticker:        config.Ticker, // если не передан (nil) - создадим в startPeriodicSave
	}

	// Загружаем данные из файла если нужно
	if config.Restore {
		if err := fs.loadFromFile(); err != nil {
			fs.logger.Error("Ошибка при загрузке метрик из файла", zap.String("file", config.FilePath), zap.Error(err))
		} else {
			fs.logger.Info("Метрики успешно загружены из файла", zap.String("file", config.FilePath))
		}
	}

	// Запускаем периодическое сохранение если интервал > 0
	if fs.storeInterval > 0 {
		fs.startPeriodicSave()
	} else {
		fs.logger.Info("Режим синхронного сохранения активирован (STORE_INTERVAL=0)")
	}

	return fs, nil
}

// UpdateMetric реализует интерфейс Storage
func (fs *FileStorage) UpdateMetric(metric models.Metrics) error {
	// Обновляем в памяти
	if err := fs.mem.UpdateMetric(metric); err != nil {
		return err
	}

	// Синхронно сохраняем в файл если нужно
	if fs.syncWrite {
		return fs.saveToFile()
	}

	return nil
}

// GetMetric реализует интерфейс Storage
func (fs *FileStorage) GetMetric(id string, mType string) (models.Metrics, bool) {
	return fs.mem.GetMetric(id, mType)
}

// GetAllMetrics реализует интерфейс Storage
func (fs *FileStorage) GetAllMetrics() []models.Metrics {
	return fs.mem.GetAllMetrics()
}

// saveToFile сохраняет все метрики в JSON файл
func (fs *FileStorage) saveToFile() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Получаем все метрики
	metrics := fs.mem.GetAllMetrics()

	// Открываем файл для записи (создаём, если не существует)
	file, err := os.Create(fs.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Сохраняем в JSON с отступами для читаемости
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metrics); err != nil {
		return err
	}

	// Инкрементируем счетчик успешных сохранений
	fs.saveCount.Add(1)

	return nil
}

// loadFromFile загружает метрики из JSON файла
func (fs *FileStorage) loadFromFile() error {
	// Проверяем существование файла
	if _, err := os.Stat(fs.filePath); os.IsNotExist(err) {
		return nil // Файл не существует - это не ошибка при первом запуске
	}

	file, err := os.Open(fs.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var metrics []models.Metrics
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metrics); err != nil {
		return err
	}

	// Загружаем метрики
	for _, metric := range metrics {
		fs.mem.UpdateMetric(metric)
	}

	return nil
}

// startPeriodicSave запускает горутину для периодического сохранения
func (fs *FileStorage) startPeriodicSave() {
	// Создаем ticker если не был передан
	if fs.ticker == nil {
		fs.ticker = &timeTicker{time.NewTicker(time.Duration(fs.storeInterval) * time.Second)}
	}

	fs.logger.Info("Периодическое сохранение метрик активировано", zap.Int("interval_sec", fs.storeInterval))

	go func() {
		for {
			select {
			case <-fs.ticker.C():
				if err := fs.saveToFile(); err != nil {
					fs.logger.Error("Ошибка при сохранении метрик",
						zap.String("file", fs.filePath),
						zap.Int64("total_saves", fs.saveCount.Load()),
						zap.Error(err),
					)
				} else {
					fs.logger.Debug("Метрики успешно сохранены",
						zap.String("file", fs.filePath),
						zap.Int64("total_saves", fs.saveCount.Load()),
					)
				}
			case <-fs.stopChan:
				return
			}
		}
	}()
}

// Close завершает работу FileStorage и сохраняет данные
func (fs *FileStorage) Close() error {
	// Останавливаем периодическое сохранение
	if fs.ticker != nil {
		fs.ticker.Stop()
	}
	close(fs.stopChan)

	// Финальное сохранение
	if err := fs.saveToFile(); err != nil {
		fs.logger.Error("Ошибка при финальном сохранении метрик",
			zap.String("file", fs.filePath),
			zap.Int64("total_saves", fs.saveCount.Load()),
			zap.Error(err),
		)
		return err
	}

	fs.logger.Info("Метрики успешно сохранены перед выходом",
		zap.String("file", fs.filePath),
		zap.Int64("total_saves", fs.saveCount.Load()),
	)
	return nil
}

// GetSaveCount возвращает общее количество операций сохранения
func (fs *FileStorage) GetSaveCount() int64 {
	return fs.saveCount.Load()
}

// Убедимся, что FileStorage удовлетворяет интерфейсу Storage
var _ Storage = (*FileStorage)(nil)
