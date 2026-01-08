package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/paxren/metrics/internal/models"
)

func TestURLObserver_Notify(t *testing.T) {
	// Создаём тестовый сервер
	receivedEvents := make([]*models.AuditEvent, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event models.AuditEvent
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receivedEvents = append(receivedEvents, &event)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаём наблюдателя
	observer := NewURLObserver(server.URL)

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

	// Проверяем, что событие было получено
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

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	}

	// Уведомляем наблюдателя - должна быть ошибка
	err := observer.Notify(event)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
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

	// Проверяем, что все события были получены
	mu.Lock()
	defer mu.Unlock()

	expectedEvents := numGoroutines * numEvents
	if len(receivedEvents) != expectedEvents {
		t.Errorf("Expected %d events, got %d", expectedEvents, len(receivedEvents))
	}
}
