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

type URLObserver struct {
	url       string
	client    *http.Client
	eventChan chan *models.AuditEvent
	done      chan struct{}
	wg        sync.WaitGroup
}

func NewURLObserver(url string) *URLObserver {
	return NewURLObserverWithBufferSize(url, 100) // Буфер по умолчанию
}

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

func (u *URLObserver) Notify(event *models.AuditEvent) error {
	select {
	case u.eventChan <- event:
		return nil
	default:
		// Канал переполнен, логируем и возвращаем ошибку
		return errors.New("url observer audit queue is full")
	}
}

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

func (u *URLObserver) Close() error {
	close(u.done)
	u.wg.Wait()
	return nil
}
