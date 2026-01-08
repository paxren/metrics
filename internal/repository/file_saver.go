package repository

import (
	"fmt"
	"os"
	"time"

	"encoding/json"

	"github.com/paxren/metrics/internal/models"
)

// FileSaver реализует сохранение метрик в файл с периодическим автоматическим сохранением.
//
// Оборачивает другое хранилище (Repository) и добавляет функциональность
// сохранения всех метрик в файл в формате JSON.
// Поддерживает автоматическое сохранение с заданным интервалом.
type FileSaver struct {
	saver  *os.File
	ticker *time.Ticker
	Repository
	fileName string
}

// MakeSavedRepo создаёт новое хранилище с поддержкой сохранения в файл.
//
// Если интервал больше 0, запускается горутина для периодического сохранения.
//
// Параметры:
//   - repo: базовое хранилище метрик
//   - fileName: имя файла для сохранения метрик
//   - interval: интервал автоматического сохранения в секундах (0 - отключить)
//
// Возвращает:
//   - *FileSaver: указатель на созданное хранилище
//
// Пример использования:
//
//	storage := MakeMemStorage()
//	fileStorage := MakeSavedRepo(storage, "metrics.json", 300) // сохранять каждые 5 минут
func MakeSavedRepo(repo Repository, fileName string, interval uint64) *FileSaver {

	var ticker *time.Ticker = nil
	if interval != 0 {
		ticker = time.NewTicker(time.Duration(interval) * time.Second)
	}

	fs := &FileSaver{
		saver:      nil,
		ticker:     ticker,
		Repository: repo,
		fileName:   fileName,
	}

	if fs.ticker != nil {
		go fs.saveOnTicker()
	}

	return fs
}

// saveOnTicker запускает горутину для периодического сохранения метрик.
//
// Работает до остановки ticker.
func (fs *FileSaver) saveOnTicker() {
	for range fs.ticker.C {
		//тут не обрабатываются ошибки сейва
		fs.Save()
	}
}

// Load загружает метрики из файла в базовое хранилище.
//
// Читает файл в формате JSON и восстанавливает метрики в хранилище.
// Существующие метрики с теми же именами будут перезаписаны.
//
// Параметры:
//   - fileName: имя файла для загрузки метрик
//
// Возвращает:
//   - error: ошибка при загрузке, если она произошла
func (fs *FileSaver) Load(fileName string) error {

	data, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	metrics := make([]models.Metrics, 0)

	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			fs.Repository.UpdateGauge(metric.ID, *metric.Value)
		case models.Counter:
			fs.Repository.UpdateCounter(metric.ID, *metric.Delta)
		default:
			//переделать на просто запись ошибки, а не прекращение работы
			return fmt.Errorf("неизвестный тип метрики: %s", metric.MType)
		}
	}

	return nil
}

// Save сохраняет все метрики из базового хранилища в файл.
//
// Сохраняет метрики в формате JSON с отступами для читаемости.
// Если файл существует, он будет перезаписан.
//
// Возвращает:
//   - error: ошибка при сохранении, если она произошла
func (fs *FileSaver) Save() error {
	fw, err := os.Create(fs.fileName)
	if err != nil {
		return err
	}
	defer fw.Close()

	metrics := make([]models.Metrics, 0)

	for _, key := range fs.Repository.GetGaugesKeys() {

		value, err := fs.Repository.GetGauge(key)
		if err != nil {
			return err
		}

		metric := models.Metrics{
			ID:    key,
			MType: models.Gauge,
			Value: &value,
		}

		metrics = append(metrics, metric)
	}

	for _, key := range fs.Repository.GetCountersKeys() {

		value, err := fs.Repository.GetCounter(key)
		if err != nil {
			return err
		}

		metric := models.Metrics{
			ID:    key,
			MType: models.Counter,
			Delta: &value,
		}

		metrics = append(metrics, metric)
	}

	data, err := json.MarshalIndent(&metrics, "", "\t")
	if err != nil {
		return err
	}

	_, err = fw.Write(data)
	if err != nil {
		return err
	}

	fmt.Println("saved")
	return nil

}

// UpdateGauge обновляет или создаёт метрику типа gauge с указанным именем и значением.
//
// Если автоматическое сохранение отключено (interval = 0), выполняет немедленное сохранение.
//
// Параметры:
//   - key: имя метрики
//   - value: новое значение метрики
//
// Возвращает:
//   - error: ошибка базового хранилища, если она произошла
func (fs *FileSaver) UpdateGauge(key string, value float64) error {

	err := fs.Repository.UpdateGauge(key, value)

	if fs.ticker == nil {
		//тут не обрабатываются ошибки сейва
		fs.Save()
	}

	return err
}

// UpdateCounter обновляет или создаёт метрику типа counter, добавляя указанное значение к текущему.
//
// Если автоматическое сохранение отключено (interval = 0), выполняет немедленное сохранение.
//
// Параметры:
//   - key: имя метрики
//   - value: значение, которое нужно добавить к текущему
//
// Возвращает:
//   - error: ошибка базового хранилища, если она произошла
func (fs *FileSaver) UpdateCounter(key string, value int64) error {

	err := fs.Repository.UpdateCounter(key, value)
	if fs.ticker == nil {
		//тут не обрабатываются ошибки сейва
		fs.Save()
	}

	return err
}
