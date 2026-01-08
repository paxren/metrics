package audit

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/paxren/metrics/internal/models"
)

type FileObserver struct {
	filePath  string
	eventChan chan *models.AuditEvent
	done      chan struct{}
	wg        sync.WaitGroup
}

func NewFileObserver(filePath string) *FileObserver {
	return NewFileObserverWithBufferSize(filePath, 100) // Буфер по умолчанию
}

func NewFileObserverWithBufferSize(filePath string, bufferSize int) *FileObserver {
	f := &FileObserver{
		filePath:  filePath,
		eventChan: make(chan *models.AuditEvent, bufferSize),
		done:      make(chan struct{}),
	}

	f.wg.Add(1)
	go f.processEvents()

	return f
}

func (f *FileObserver) Notify(event *models.AuditEvent) error {
	select {
	case f.eventChan <- event:
		return nil
	default:
		// Канал переполнен, логируем и возвращаем ошибку
		return errors.New("file observer audit queue is full")
	}
}

func (f *FileObserver) processEvents() {
	defer f.wg.Done()

	for {
		select {
		case event := <-f.eventChan:
			f.writeToFile(event)
		case <-f.done:
			// Обрабатываем оставшиеся события перед выходом
			for len(f.eventChan) > 0 {
				f.writeToFile(<-f.eventChan)
			}
			return
		}
	}
}

func (f *FileObserver) writeToFile(event *models.AuditEvent) {
	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return
	}

	_, err = file.Write(append(data, '\n'))
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return
	}
}

func (f *FileObserver) Close() error {
	close(f.done)
	f.wg.Wait()
	return nil
}
