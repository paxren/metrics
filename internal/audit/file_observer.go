package audit

import (
	"encoding/json"
	"os"

	"github.com/paxren/metrics/internal/models"
)

// FileObserver реализует наблюдателя, который записывает события аудита в файл.
//
// Использует BaseObserver для управления очередью событий и асинхронной обработки.
// Каждое событие записывается в файл в формате JSON с новой строки.
type FileObserver struct {
	*BaseObserver
	filePath string
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
func NewFileObserverWithBufferSize(filePath string, bufferSize int) *FileObserver {
	handler := &fileHandler{filePath: filePath}
	return &FileObserver{
		BaseObserver: NewBaseObserver(bufferSize, handler),
		filePath:     filePath,
	}
}

// fileHandler реализует EventHandler для записи событий в файл
type fileHandler struct {
	filePath string
}

// Handle записывает событие в файл в формате JSON.
//
// Открывает файл в режиме добавления, сериализует событие и записывает его.
// В случае ошибки молча завершается (в реальном приложении нужно логирование).
//
// Параметры:
//   - event: событие аудита для записи
func (h *fileHandler) Handle(event *models.AuditEvent) {
	file, err := os.OpenFile(h.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
