package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/paxren/metrics/internal/models"
)

// URLObserver реализует наблюдателя, который отправляет события аудита на удалённый URL.
//
// Использует буферизированный канал для асинхронной отправки событий.
// Каждое событие сериализуется в JSON и отправляется POST-запросом.
type URLObserver struct {
	url       string
	client    *http.Client
	eventChan chan *models.AuditEvent
	done      chan struct{}
	wg        sync.WaitGroup
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
// Запускает горутину для асинхронной обработки событий.
// Создаёт HTTP-клиент с таймаутом 5 секунд.
//
// Параметры:
//   - url: URL для отправки событий аудита
//   - bufferSize: размер буфера для событий
//
// Возвращает:
//   - *URLObserver: указатель на созданного наблюдателя
func NewURLObserverWithBufferSize(url string, bufferSize int) *URLObserver {
	u := &URLObserver{
		url:       url,
		eventChan: make(chan *models.AuditEvent, bufferSize),
		done:      make(chan struct{}),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	u.wg.Add(1)
	go u.processEvents()

	return u
}

// Notify отправляет событие в канал для обработки.
//
// Если канал переполнен, возвращает ошибку.
//
// Параметры:
//   - event: событие аудита для отправки
//
// Возвращает:
//   - error: ошибка если канал переполнен
func (u *URLObserver) Notify(event *models.AuditEvent) error {
	select {
	case u.eventChan <- event:
		return nil
	default:
		// Канал переполнен, логируем и возвращаем ошибку
		return errors.New("url observer audit queue is full")
	}
}

// processEvents обрабатывает события из канала в отдельной горутине.
//
// Отправляет события на URL до получения сигнала завершения.
func (u *URLObserver) processEvents() {
	defer u.wg.Done()

	for {
		select {
		case event := <-u.eventChan:
			u.sendToURL(event)
		case <-u.done:
			// Обрабатываем оставшиеся события перед выходом
			for len(u.eventChan) > 0 {
				u.sendToURL(<-u.eventChan)
			}
			return
		}
	}
}

// sendToURL отправляет событие на удалённый URL.
//
// Сериализует событие в JSON и отправляет POST-запросом.
// В случае ошибки молча завершается (в реальном приложении нужно логирование).
//
// Параметры:
//   - event: событие аудита для отправки
func (u *URLObserver) sendToURL(event *models.AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return
	}

	resp, err := u.client.Post(u.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		// В реальном приложении здесь должно быть логирование ошибки
		return
	}
	defer resp.Body.Close()
}

// Close останавливает обработку событий и ожидает завершения горутины.
//
// Возвращает:
//   - error: всегда nil
func (u *URLObserver) Close() error {
	close(u.done)
	u.wg.Wait()
	return nil
}
