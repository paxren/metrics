package audit

import (
	"sync"
	"testing"
	"time"

	"github.com/paxren/metrics/internal/models"
)

// mockHandler реализует EventHandler для тестирования
type mockHandler struct {
	events []*models.AuditEvent
	mu     sync.Mutex
}

func (h *mockHandler) Handle(event *models.AuditEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
	return nil
}

func (h *mockHandler) getEvents() []*models.AuditEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.events
}

func (h *mockHandler) countEvents() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.events)
}

func TestBaseObserver_Notify(t *testing.T) {
	handler := &mockHandler{}
	observer := NewBaseObserver(10, handler)
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

	// Проверяем, что событие было обработано
	if handler.countEvents() != 1 {
		t.Fatalf("Expected 1 event, got %d", handler.countEvents())
	}

	receivedEvent := handler.getEvents()[0]
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

func TestBaseObserver_ConcurrentAccess(t *testing.T) {
	handler := &mockHandler{}
	observer := NewBaseObserver(100, handler)
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

	// Проверяем, что все события были обработаны
	expectedEvents := numGoroutines * numEvents
	if handler.countEvents() != expectedEvents {
		t.Errorf("Expected %d events, got %d", expectedEvents, handler.countEvents())
	}
}

func TestBaseObserver_QueueOverflow(t *testing.T) {
	handler := &mockHandler{}
	observer := NewBaseObserver(2, handler)
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

func TestBaseObserver_Close(t *testing.T) {
	handler := &mockHandler{}
	observer := NewBaseObserver(10, handler)

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

	// Проверяем, что событие было обработано
	if handler.countEvents() != 1 {
		t.Errorf("Expected 1 event after close, got %d", handler.countEvents())
	}
}

func TestBaseObserver_CloseWithPendingEvents(t *testing.T) {
	handler := &mockHandler{}
	observer := NewBaseObserver(10, handler)

	// Создаём несколько тестовых событий
	events := make([]*models.AuditEvent, 3)
	for i := 0; i < 3; i++ {
		events[i] = &models.AuditEvent{
			TS:        int64(1234567890 + i),
			Metrics:   []string{"Alloc"},
			IPAddress: "192.168.0.42",
		}

		// Уведомляем наблюдателя
		err := observer.Notify(events[i])
		if err != nil {
			t.Fatalf("Failed to notify observer: %v", err)
		}
	}

	// Закрываем наблюдателя без ожидания
	err := observer.Close()
	if err != nil {
		t.Fatalf("Failed to close observer: %v", err)
	}

	// Ждем некоторое время для обработки оставшихся событий
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что все события были обработаны
	if handler.countEvents() != 3 {
		t.Errorf("Expected 3 events after close, got %d", handler.countEvents())
	}
}

func TestBaseObserver_NotifyAfterClose(t *testing.T) {
	handler := &mockHandler{}
	observer := NewBaseObserver(10, handler)

	// Закрываем наблюдателя
	err := observer.Close()
	if err != nil {
		t.Fatalf("Failed to close observer: %v", err)
	}

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc"},
		IPAddress: "192.168.0.42",
	}

	// Попытка уведомить после закрытия должна вызвать панику,
	// так как канал eventChan закрыт
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when notifying after close")
		}
	}()

	observer.Notify(event)
}
