package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/paxren/metrics/internal/models"
)

// URLObserver реализует наблюдателя, который отправляет события аудита на удалённый URL.
//
// Использует BaseObserver для управления очередью событий и асинхронной обработки.
// Каждое событие сериализуется в JSON и отправляется POST-запросом.
type URLObserver struct {
	*BaseObserver
	url    string
	client *http.Client
}

// NewURLObserver создаёт новый наблюдатель с буфером по умолчанию (100 событий).
//
// Параметры:
//   - url: URL для отправки событий аудита
//
// Возвращает:
//   - *URLObserver: указатель на созданного наблюдателя
//
// Пример использования:
//
//	observer := NewURLObserver("https://example.com/audit")
//	defer observer.Close()
//	event := NewAuditEvent([]string{"metric1"}, "192.168.1.1")
//	observer.Notify(event)
func NewURLObserver(url string) *URLObserver {
	return NewURLObserverWithBufferSize(url, 100) // Буфер по умолчанию
}

// NewURLObserverWithBufferSize создаёт новый наблюдатель с указанным размером буфера.
//
// Параметры:
//   - url: URL для отправки событий аудита
//   - bufferSize: размер буфера для событий
//
// Возвращает:
//   - *URLObserver: указатель на созданного наблюдателя
func NewURLObserverWithBufferSize(url string, bufferSize int) *URLObserver {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	handler := &urlHandler{
		url:    url,
		client: client,
	}

	return &URLObserver{
		BaseObserver: NewBaseObserver(bufferSize, handler),
		url:          url,
		client:       client,
	}
}

// urlHandler реализует EventHandler для отправки событий на удалённый URL
type urlHandler struct {
	url    string
	client *http.Client
}

// Handle отправляет событие на удалённый URL.
//
// Сериализует событие в JSON и отправляет POST-запросом.
// В случае ошибки молча завершается (в реальном приложении нужно логирование).
//
// Параметры:
//   - event: событие аудита для отправки
func (h *urlHandler) Handle(event *models.AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return
	}
	defer resp.Body.Close()
}
