package audit

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/paxren/metrics/internal/models"
)

// FileObserver реализует наблюдателя, который записывает события аудита в файл.
//
// Использует буферизированный канал для асинхронной записи событий.
// Каждое событие записывается в файл в формате JSON с новой строки.
type FileObserver struct {
	filePath  string
	eventChan chan *models.AuditEvent
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewFileObserver создаёт новый наблюдатель с буфером по умолчанию (100 событий).
//
// Параметры:
//   - filePath: путь к файлу для записи событий аудита
//
// Возвращает:
//   - *FileObserver: указатель на созданного наблюдателя
//
// Пример использования:
//
//	observer := NewFileObserver("audit.log")
//	defer observer.Close()
//	event := NewAuditEvent([]string{"metric1"}, "192.168.1.1")
//	observer.Notify(event)
func NewFileObserver(filePath string) *FileObserver {
	return NewFileObserverWithBufferSize(filePath, 100) // Буфер по умолчанию
}

// NewFileObserverWithBufferSize создаёт новый наблюдатель с указанным размером буфера.
//
// Запускает горутину для асинхронной обработки событий.
//
// Параметры:
//   - filePath: путь к файлу для записи событий аудита
//   - bufferSize: размер буфера для событий
//
// Возвращает:
//   - *FileObserver: указатель на созданного наблюдателя
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

// Notify отправляет событие в канал для обработки.
//
// Если канал переполнен, возвращает ошибку.
//
// Параметры:
//   - event: событие аудита для записи
//
// Возвращает:
//   - error: ошибка если канал переполнен
func (f *FileObserver) Notify(event *models.AuditEvent) error {
	select {
	case f.eventChan <- event:
		return nil
	default:
		// Канал переполнен, логируем и возвращаем ошибку
		return errors.New("file observer audit queue is full")
	}
}

// processEvents обрабатывает события из канала в отдельной горутине.
//
// Записывает события в файл до получения сигнала завершения.
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

// writeToFile записывает событие в файл в формате JSON.
//
// Открывает файл в режиме добавления, сериализует событие и записывает его.
// В случае ошибки молча завершается (в реальном приложении нужно логирование).
//
// Параметры:
//   - event: событие аудита для записи
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

// Close останавливает обработку событий и ожидает завершения горутины.
//
// Возвращает:
//   - error: всегда nil
func (f *FileObserver) Close() error {
	close(f.done)
	f.wg.Wait()
	return nil
}
