package audit

import (
	"errors"
	"fmt"
	"sync"

	"github.com/paxren/metrics/internal/models"
)

// Observer определяет интерфейс для наблюдателей за событиями аудита.
//
// Наблюдатели получают уведомления о событиях аудита и могут
// обрабатывать их различными способами: запись в файл, отправка
// на удалённый сервер и т.д.
type Observer interface {
	// Notify обрабатывает событие аудита.
	//
	// Параметры:
	//   - event: событие аудита для обработки
	//
	// Возвращает:
	//   - error: ошибка при обработке события, если она произошла
	Notify(event *models.AuditEvent) error

	// Close закрывает наблюдатель и освобождает ресурсы.
	//
	// Должен вызываться при завершении работы с наблюдателем.
	//
	// Возвращает:
	//   - error: ошибка при закрытии, если она произошла
	Close() error
}

// EventHandler определяет интерфейс для обработки событий аудита.
//
// Реализации этого интерфейса инкапсулируют специфическую логику
// обработки событий (запись в файл, отправка по сети и т.д.).
type EventHandler interface {
	// Handle обрабатывает событие аудита.
	//
	// Параметры:
	//   - event: событие аудита для обработки
	//
	// Возвращает:
	//   - error: ошибка при обработке события, если она произошла
	Handle(event *models.AuditEvent) error
}

// BaseObserver содержит общую логику для всех наблюдателей.
//
// Управляет очередью событий, асинхронной обработкой и корректным
// завершением работы. Использует встраивание в конкретные реализации
// наблюдателей для избежания дублирования кода.
type BaseObserver struct {
	eventChan chan *models.AuditEvent
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewBaseObserver создаёт базового наблюдателя с указанным обработчиком.
//
// Запускает горутину для асинхронной обработки событий.
//
// Параметры:
//   - bufferSize: размер буфера для событий
//   - handler: обработчик событий
//
// Возвращает:
//   - *BaseObserver: указатель на созданного наблюдателя
func NewBaseObserver(bufferSize int, handler EventHandler) *BaseObserver {
	b := &BaseObserver{
		eventChan: make(chan *models.AuditEvent, bufferSize),
		done:      make(chan struct{}),
	}

	b.wg.Add(1)
	go b.processEvents(handler)

	return b
}

// processEvents обрабатывает события из канала в отдельной горутине.
//
// Делегирует обработку событий указанному обработчику.
// Обрабатывает оставшиеся события перед выходом.
// В случае ошибки обработки события логирует её (в реальном приложении).
//
// Параметры:
//   - handler: обработчик событий
func (b *BaseObserver) processEvents(handler EventHandler) {
	defer b.wg.Done()

	for {
		select {
		case event := <-b.eventChan:
			if err := handler.Handle(event); err != nil {
				fmt.Printf("Error handling audit event: %v\n", err)
			}
		case <-b.done:
			// Обрабатываем оставшиеся события перед выходом
			for len(b.eventChan) > 0 {
				if err := handler.Handle(<-b.eventChan); err != nil {
					fmt.Printf("Error handling remaining audit event: %v\n", err)
				}
			}
			return
		}
	}
}

// Notify отправляет событие в канал для обработки.
//
// Если канал переполнен, возвращает ошибку.
//
// Параметры:
//   - event: событие аудита для обработки
//
// Возвращает:
//   - error: ошибка если канал переполнен
func (b *BaseObserver) Notify(event *models.AuditEvent) error {
	select {
	case b.eventChan <- event:
		return nil
	default:
		// Канал переполнен, возвращаем ошибку
		return errors.New("audit queue is full")
	}
}

// Close останавливает обработку событий и ожидает завершения горутины.
//
// Возвращает:
//   - error: всегда nil
func (b *BaseObserver) Close() error {
	close(b.done)
	b.wg.Wait()
	close(b.eventChan)
	return nil
}
