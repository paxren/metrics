package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestNewLogger проверяет создание нового экземпляра Logger
func TestNewLogger(t *testing.T) {
	// Создаем тестовый logger
	zapLogger := zap.NewNop()
	logger := NewLogger(zapLogger)

	// Проверяем, что logger не nil
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Проверяем, что поля инициализированы правильно
	if logger.logger != zapLogger {
		t.Errorf("Expected logger.logger to be the provided zap logger")
	}

	if logger.sugar == nil {
		t.Error("Expected logger.sugar to be initialized")
	}
}

// TestWithLogging проверяет работу middleware для логирования
func TestWithLogging(t *testing.T) {
	// Создаем наблюдаемый logger для перехвата логов
	zapCore, logs := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(zapCore)
	logger := NewLogger(zapLogger)

	// Создаем тестовый обработчик, который возвращает определенный статус и тело
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
	}

	// Оборачиваем тестовый обработчик нашим middleware
	wrappedHandler := logger.WithLogging(testHandler)

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/test/path?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Accept", "application/json")

	// Создаем ResponseRecorder для захвата ответа
	rr := httptest.NewRecorder()

	// Вызываем обернутый обработчик
	wrappedHandler.ServeHTTP(rr, req)

	// Проверяем, что ответ корректен
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedBody := "test response"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, body)
	}

	// Проверяем, что был записан лог
	if logs.Len() != 1 {
		t.Errorf("Expected 1 log entry, got %d", logs.Len())
	}

	// Проверяем содержимое лога - так как используется Infoln, проверяем наличие ключевых слов в сообщении
	if logs.Len() > 0 {
		logEntry := logs.All()[0]
		message := logEntry.Message

		// Проверяем, что в сообщении содержатся ожидаемые значения
		if !strings.Contains(message, "uri") {
			t.Error("Expected 'uri' to be present in log message")
		}
		if !strings.Contains(message, "method") {
			t.Error("Expected 'method' to be present in log message")
		}
		if !strings.Contains(message, "status") {
			t.Error("Expected 'status' to be present in log message")
		}
		if !strings.Contains(message, "duration") {
			t.Error("Expected 'duration' to be present in log message")
		}
		if !strings.Contains(message, "size") {
			t.Error("Expected 'size' to be present in log message")
		}
		if !strings.Contains(message, "requestHeaders") {
			t.Error("Expected 'requestHeaders' to be present in log message")
		}
		if !strings.Contains(message, "responceHeaders") {
			t.Error("Expected 'responceHeaders' to be present in log message")
		}

		// Проверяем наличие конкретных значений в сообщении
		if !strings.Contains(message, "/test/path?param=value") {
			t.Error("Expected URI to be present in log message")
		}
		if !strings.Contains(message, "GET") {
			t.Error("Expected method to be present in log message")
		}
	}
}

// TestWithLoggingWithErrorStatus проверяет логирование ответов с ошибочными статусами
func TestWithLoggingWithErrorStatus(t *testing.T) {
	zapCore, logs := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(zapCore)
	logger := NewLogger(zapLogger)

	// Создаем обработчик, который возвращает ошибку
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}

	wrappedHandler := logger.WithLogging(testHandler)

	req := httptest.NewRequest("POST", "/error", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Проверяем, что статус ошибки корректно залогирован
	if logs.Len() != 1 {
		t.Errorf("Expected 1 log entry, got %d", logs.Len())
	}

	if logs.Len() > 0 {
		logEntry := logs.All()[0]
		message := logEntry.Message

		// Проверяем наличие ключевых слов в сообщении
		if !strings.Contains(message, "POST") {
			t.Error("Expected method 'POST' to be present in log message")
		}
		if !strings.Contains(message, "/error") {
			t.Error("Expected URI '/error' to be present in log message")
		}
	}
}

// TestWithLoggingWithEmptyResponse проверяет логирование пустых ответов
func TestWithLoggingWithEmptyResponse(t *testing.T) {
	zapCore, logs := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(zapCore)
	logger := NewLogger(zapLogger)

	// Создаем обработчик, который не пишет тело ответа
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// Не пишем тело ответа
	}

	wrappedHandler := logger.WithLogging(testHandler)

	req := httptest.NewRequest("DELETE", "/resource/123", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Проверяем, что пустой ответ корректно залогирован
	if logs.Len() != 1 {
		t.Errorf("Expected 1 log entry, got %d", logs.Len())
	}

	if logs.Len() > 0 {
		logEntry := logs.All()[0]
		message := logEntry.Message

		// Проверяем наличие ключевых слов в сообщении
		if !strings.Contains(message, "DELETE") {
			t.Error("Expected method 'DELETE' to be present in log message")
		}
		if !strings.Contains(message, "/resource/123") {
			t.Error("Expected URI '/resource/123' to be present in log message")
		}
	}
}

// TestLoggingResponseWriterWrite проверяет метод Write loggingResponseWriter
func TestLoggingResponseWriterWrite(t *testing.T) {
	// Создаем ResponseRecorder для захвата ответа
	rr := httptest.NewRecorder()

	// Создаем responseData для отслеживания размера
	responseData := &responseData{
		status:  0,
		size:    0,
		headers: nil,
	}

	// Создаем loggingResponseWriter
	lrw := &loggingResponseWriter{
		ResponseWriter: rr,
		responseData:   responseData,
	}

	// Тестируем запись данных
	testData := []byte("test data")
	n, err := lrw.Write(testData)

	// Проверяем результат записи
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Проверяем, что размер был обновлен
	if lrw.responseData.size != len(testData) {
		t.Errorf("Expected response size %d, got %d", len(testData), lrw.responseData.size)
	}

	// Проверяем, что данные были записаны в оригинальный ResponseWriter
	if rr.Body.String() != string(testData) {
		t.Errorf("Expected body %q, got %q", string(testData), rr.Body.String())
	}
}

// TestLoggingResponseWriterWriteMultiple проверяет множественные вызовы Write
func TestLoggingResponseWriterWriteMultiple(t *testing.T) {
	rr := httptest.NewRecorder()

	responseData := &responseData{
		status:  0,
		size:    0,
		headers: nil,
	}

	lrw := &loggingResponseWriter{
		ResponseWriter: rr,
		responseData:   responseData,
	}

	// Делаем несколько записей
	data1 := []byte("first ")
	data2 := []byte("second ")
	data3 := []byte("third")

	lrw.Write(data1)
	lrw.Write(data2)
	lrw.Write(data3)

	expectedSize := len(data1) + len(data2) + len(data3)
	expectedBody := string(data1) + string(data2) + string(data3)

	// Проверяем, что размер накоплен правильно
	if lrw.responseData.size != expectedSize {
		t.Errorf("Expected total size %d, got %d", expectedSize, lrw.responseData.size)
	}

	// Проверяем, что тело ответа корректно
	if rr.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rr.Body.String())
	}
}

// TestLoggingResponseWriterWriteHeader проверяет метод WriteHeader loggingResponseWriter
func TestLoggingResponseWriterWriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()

	responseData := &responseData{
		status:  0,
		size:    0,
		headers: nil,
	}

	lrw := &loggingResponseWriter{
		ResponseWriter: rr,
		responseData:   responseData,
	}

	// Тестируем установку статуса
	testStatus := http.StatusCreated
	lrw.WriteHeader(testStatus)

	// Проверяем, что статус был сохранен
	if lrw.responseData.status != testStatus {
		t.Errorf("Expected status %d, got %d", testStatus, lrw.responseData.status)
	}

	// Проверяем, что статус был установлен в оригинальном ResponseWriter
	if rr.Code != testStatus {
		t.Errorf("Expected ResponseWriter status %d, got %d", testStatus, rr.Code)
	}
}

// TestLoggingResponseWriterHeader проверяет метод Header loggingResponseWriter
func TestLoggingResponseWriterHeader(t *testing.T) {
	rr := httptest.NewRecorder()

	responseData := &responseData{
		status:  0,
		size:    0,
		headers: nil,
	}

	lrw := &loggingResponseWriter{
		ResponseWriter: rr,
		responseData:   responseData,
	}

	// Получаем заголовки и устанавливаем тестовые значения
	headers := lrw.Header()
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Custom-Header", "test-value")

	// Проверяем, что заголовки были сохранены в responseData
	if lrw.responseData.headers == nil {
		t.Error("Expected headers to be captured in responseData")
	} else {
		if lrw.responseData.headers.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type %q, got %q", "application/json", lrw.responseData.headers.Get("Content-Type"))
		}
		if lrw.responseData.headers.Get("X-Custom-Header") != "test-value" {
			t.Errorf("Expected X-Custom-Header %q, got %q", "test-value", lrw.responseData.headers.Get("X-Custom-Header"))
		}
	}

	// Проверяем, что заголовки были установлены в оригинальном ResponseWriter
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected ResponseWriter Content-Type %q, got %q", "application/json", rr.Header().Get("Content-Type"))
	}
}

// TestWithLoggingPanic проверяет обработку паники в обработчике
func TestWithLoggingPanic(t *testing.T) {
	zapCore, _ := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(zapCore)
	logger := NewLogger(zapLogger)

	// Создаем обработчик, который вызывает панику
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}

	wrappedHandler := logger.WithLogging(testHandler)

	req := httptest.NewRequest("GET", "/panic", nil)
	rr := httptest.NewRecorder()

	// Проверяем, что паника не обрабатывается нашим middleware (она должна распространяться)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be propagated")
		} else if r != "test panic" {
			t.Errorf("Expected panic message 'test panic', got %v", r)
		}
	}()

	wrappedHandler.ServeHTTP(rr, req)
}

// BenchmarkWithLogging проверяет производительность middleware
func BenchmarkWithLogging(b *testing.B) {
	zapLogger := zap.NewNop()
	logger := NewLogger(zapLogger)

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	wrappedHandler := logger.WithLogging(testHandler)
	req := httptest.NewRequest("GET", "/benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}
