package models

import "time"

// AuditEvent представляет событие аудита, которое записывается при изменении метрик.
//
// Содержит информацию о времени события, измененных метриках и IP-адресе клиента.
// Используется для отслеживания и анализа изменений в системе метрик.
//
// Поля:
//   - TS: временная метка события в формате Unix timestamp
//   - Metrics: список имён измененных метрик
//   - IPAddress: IP-адрес клиента, инициировавшего изменение
type AuditEvent struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// NewAuditEvent создаёт новое событие аудита с указанными метриками и IP-адресом.
//
// Автоматически устанавливает временную метку текущего времени.
//
// Параметры:
//   - metrics: список имён измененных метрик
//   - ipAddress: IP-адрес клиента, инициировавшего изменение
//
// Возвращает:
//   - *AuditEvent: указатель на созданное событие аудита
//
// Пример использования:
//
//	event := NewAuditEvent([]string{"alloc", "gc"}, "192.168.1.1")
//	fmt.Printf("Event created at %d for metrics %v", event.TS, event.Metrics)
func NewAuditEvent(metrics []string, ipAddress string) *AuditEvent {
	return &AuditEvent{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
}
