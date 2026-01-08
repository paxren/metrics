package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/paxren/metrics/internal/models"
)

func TestURLObserver_Notify(t *testing.T) {
	// Создаём тестовый сервер
	receivedEvents := make([]*models.AuditEvent, 0)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event models.AuditEvent
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedEvents = append(receivedEvents, &event)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаём наблюдателя
	observer := NewURLObserver(server.URL)
	defer observer.Close()

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	}

	// Уведомляем наблюдателя
	err := observer.Notify(event)
	if err != nil {
		t.Fatalf("Failed to notify observer: %v", err)
	}

	// Ждем некоторое время для асинхронной обработки
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что событие было получено
	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(receivedEvents))
	}

	receivedEvent := receivedEvents[0]
	if receivedEvent.TS != event.TS {
		t.Errorf("Expected TS %d, got %d", event.TS, receivedEvent.TS)
	}

	if len(receivedEvent.Metrics) != len(event.Metrics) {
		t.Errorf("Expected %d metrics, got %d", len(event.Metrics), len(receivedEvent.Metrics))
	}

	if receivedEvent.IPAddress != event.IPAddress {
		t.Errorf("Expected IP %s, got %s", event.IPAddress, receivedEvent.IPAddress)
	}
}

func TestURLObserver_ServerError(t *testing.T) {
	// Создаём тестовый сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Создаём наблюдателя
	observer := NewURLObserver(server.URL)
	defer observer.Close()

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	}

	// Уведомляем наблюдателя - не должно быть ошибки даже при 500 статусе
	err := observer.Notify(event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestURLObserver_InvalidURL(t *testing.T) {
	// Создаём наблюдателя с невалидным URL
	observer := NewURLObserver("http://invalid-url-that-does-not-exist.local")
	defer observer.Close()

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	}

	// Уведомляем наблюдателя - не должно быть ошибки (асинхронная обработка)
	err := observer.Notify(event)
	if err != nil {
		t.Errorf("Unexpected error for invalid URL (async processing): %v", err)
	}
}

func TestURLObserver_ConcurrentAccess(t *testing.T) {
	// Создаём тестовый сервер
	receivedEvents := make([]*models.AuditEvent, 0)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event models.AuditEvent
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedEvents = append(receivedEvents, &event)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаём наблюдателя
	observer := NewURLObserver(server.URL)
	defer observer.Close()

	// Количество горутин для конкурентной записи
	numGoroutines := 10
	numEvents := 5

	done := make(chan bool, numGoroutines)

	// Запускаем несколько горутин для конкурентной записи
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numEvents; j++ {
				event := &models.AuditEvent{
					TS:        int64(id*100 + j),
					Metrics:   []string{string(rune('A' + id))},
					IPAddress: "192.168.0.42",
				}

				err := observer.Notify(event)
				if err != nil {
					t.Errorf("Failed to notify observer: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Ожидаем завершения всех горутин
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Ждем некоторое время для асинхронной обработки всех событий
	time.Sleep(500 * time.Millisecond)

	// Проверяем, что все события были получены
	mu.Lock()
	defer mu.Unlock()

	expectedEvents := numGoroutines * numEvents
	if len(receivedEvents) != expectedEvents {
		t.Errorf("Expected %d events, got %d", expectedEvents, len(receivedEvents))
	}
}

func TestURLObserver_QueueOverflow(t *testing.T) {
	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Имитируем медленный ответ
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаём наблюдателя с маленьким буфером
	observer := NewURLObserverWithBufferSize(server.URL, 2)
	defer observer.Close()

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc"},
		IPAddress: "192.168.0.42",
	}

	// Заполняем буфер
	err := observer.Notify(event)
	if err != nil {
		t.Fatalf("Failed to notify observer: %v", err)
	}

	err = observer.Notify(event)
	if err != nil {
		t.Fatalf("Failed to notify observer: %v", err)
	}

	// Третий вызов должен вернуть ошибку переполнения
	err = observer.Notify(event)
	if err == nil {
		t.Error("Expected queue overflow error, got nil")
	}
}

func TestURLObserver_Close(t *testing.T) {
	// Создаём тестовый сервер
	receivedEvents := make([]*models.AuditEvent, 0)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event models.AuditEvent
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedEvents = append(receivedEvents, &event)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаём наблюдателя
	observer := NewURLObserver(server.URL)

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc"},
		IPAddress: "192.168.0.42",
	}

	// Уведомляем наблюдателя
	err := observer.Notify(event)
	if err != nil {
		t.Fatalf("Failed to notify observer: %v", err)
	}

	// Закрываем наблюдателя
	err = observer.Close()
	if err != nil {
		t.Fatalf("Failed to close observer: %v", err)
	}

	// Ждем некоторое время для обработки оставшихся событий
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что событие было отправлено
	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 event after close, got %d", len(receivedEvents))
	}
}
