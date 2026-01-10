package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/paxren/metrics/internal/models"
)

// FileObserver реализует наблюдателя, который записывает события аудита в файл.
//
// Использует BaseObserver для управления очередью событий и асинхронной обработки.
// Каждое событие записывается в файл в формате JSON с новой строки.
type FileObserver struct {
	*BaseObserver
	handler *fileHandler
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
// Параметры:
//   - filePath: путь к файлу для записи событий аудита
//   - bufferSize: размер буфера для событий
//
// Возвращает:
//   - *FileObserver: указатель на созданного наблюдателя
//   - error: ошибка при создании обработчика файла
func NewFileObserverWithBufferSize(filePath string, bufferSize int) *FileObserver {
	handler, err := newFileHandler(filePath)
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return nil
	}

	return &FileObserver{
		BaseObserver: NewBaseObserver(bufferSize, handler),
		handler:      handler,
	}
}

// Close закрывает наблюдатель и освобождает ресурсы, включая файл.
//
// Возвращает:
//   - error: ошибка при закрытии, если она произошла
func (fo *FileObserver) Close() error {
	var err error

	// Сначала закрываем базовый наблюдатель, который обработает оставшиеся события в очереди
	if closeErr := fo.BaseObserver.Close(); closeErr != nil {
		err = closeErr
	}

	// Затем закрываем файл, после того как все события обработаны
	if fo.handler != nil {
		if closeErr := fo.handler.close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	return err
}

// fileHandler реализует EventHandler для записи событий в файл
type fileHandler struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
	closed   bool
}

// newFileHandler создаёт новый обработчик файла и открывает его.
//
// Параметры:
//   - filePath: путь к файлу для записи событий аудита
//
// Возвращает:
//   - *fileHandler: указатель на созданный обработчик
//   - error: ошибка при открытии файла
func newFileHandler(filePath string) (*fileHandler, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &fileHandler{
		filePath: filePath,
		file:     file,
		closed:   false,
	}, nil
}

// Handle записывает событие в файл в формате JSON.
//
// Использует предварительно открытый файл для повышения производительности.
// В случае ошибки молча завершается (в реальном приложении нужно логирование).
//
// Параметры:
//   - event: событие аудита для записи
func (h *fileHandler) Handle(event *models.AuditEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Если обработчик закрыт, возвращаем ошибку
	if h.closed {
		return fmt.Errorf("file handler is closed")
	}

	// Файл должен быть открыт (открыт в конструкторе)
	if h.file == nil {
		return fmt.Errorf("file is not open")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = h.file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write event to file: %w", err)
	}

	return nil
}

// close закрывает файл и освобождает ресурсы.
//
// Возвращает:
//   - error: ошибка при закрытии, если она произошла
func (h *fileHandler) close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Устанавливаем флаг закрытия перед закрытием файла
	h.closed = true

	if h.file != nil {
		err := h.file.Close()
		h.file = nil
		return err
	}

	return nil
}
