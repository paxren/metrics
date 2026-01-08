package audit

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/paxren/metrics/internal/models"
)

func TestFileObserver_Notify(t *testing.T) {
	// Создаём временный файл
	tmpFile, err := os.CreateTemp("", "audit_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Создаём наблюдателя
	observer := NewFileObserver(tmpFile.Name())

	// Создаём тестовое событие
	event := &models.AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	}

	// Уведомляем наблюдателя
	err = observer.Notify(event)
	if err != nil {
		t.Fatalf("Failed to notify observer: %v", err)
	}

	// Проверяем содержимое файла
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var savedEvent models.AuditEvent
	err = json.Unmarshal(data[:len(data)-1], &savedEvent) // Убираем последний символ \n
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if savedEvent.TS != event.TS {
		t.Errorf("Expected TS %d, got %d", event.TS, savedEvent.TS)
	}

	if len(savedEvent.Metrics) != len(event.Metrics) {
		t.Errorf("Expected %d metrics, got %d", len(event.Metrics), len(savedEvent.Metrics))
	}

	if savedEvent.IPAddress != event.IPAddress {
		t.Errorf("Expected IP %s, got %s", event.IPAddress, savedEvent.IPAddress)
	}
}

func TestFileObserver_ConcurrentAccess(t *testing.T) {
	// Создаём временный файл
	tmpFile, err := os.CreateTemp("", "audit_concurrent_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Создаём наблюдателя
	observer := NewFileObserver(tmpFile.Name())

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

	// Проверяем, что все события были записаны
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Подсчитываем количество строк (событий)
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}

	expectedLines := numGoroutines * numEvents
	if lines != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, lines)
	}
}
