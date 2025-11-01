package repository

import (
	"fmt"
	"os"
	"time"

	"encoding/json"

	"github.com/paxren/metrics/internal/models"
)

type FileSaver struct {
	saver  *os.File
	ticker *time.Ticker
	Repository
	fileName string
}

func MakeSavedRepo(repo Repository, fileName string, interval uint64) *FileSaver {

	//Ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)

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

func (fs *FileSaver) saveOnTicker() {
	for range fs.ticker.C {
		//тут не обрабатываются ошибки сейва
		fs.Save()
	}
}

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

	//fmt.Println(fs.repo)
	return nil
}

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

func (fs *FileSaver) UpdateGauge(key string, value float64) error {

	err := fs.Repository.UpdateGauge(key, value)

	if fs.ticker == nil {
		//тут не обрабатываются ошибки сейва
		fs.Save()
	}

	return err
}

func (fs *FileSaver) UpdateCounter(key string, value int64) error {

	err := fs.Repository.UpdateCounter(key, value)
	if fs.ticker == nil {
		//тут не обрабатываются ошибки сейва
		fs.Save()
	}

	return err
}
