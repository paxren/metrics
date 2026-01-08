package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"testing"
)

func TestWriterPool(t *testing.T) {
	pool := NewWriterPool()
	buf := &bytes.Buffer{}

	// Получаем writer
	zw1 := pool.Get(buf)
	if zw1 == nil {
		t.Fatal("Expected non-nil writer from pool")
	}

	// Пишем данные
	n, err := zw1.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 9 {
		t.Fatalf("Expected 9 bytes written, got %d", n)
	}

	// Возвращаем в пул
	pool.Put(zw1)

	// Получаем снова
	buf2 := &bytes.Buffer{}
	zw2 := pool.Get(buf2)
	if zw2 != zw1 {
		t.Error("Expected same writer instance from pool")
	}
}

func TestReaderPool(t *testing.T) {
	pool := NewReaderPool()

	// Создаем сжатые данные
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("Error writing to gzip: %v", err)
	}
	zw.Close()

	// Получаем reader
	zr1, err := pool.Get(&buf)
	if err != nil {
		t.Fatalf("Error getting reader from pool: %v", err)
	}
	if zr1 == nil {
		t.Fatal("Expected non-nil reader from pool")
	}

	// Читаем данные
	result := make([]byte, 100)
	n, err := zr1.Read(result)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}
	if string(result[:n]) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(result[:n]))
	}

	// Возвращаем в пул
	pool.Put(zr1)

	// Получаем снова - создаем новые сжатые данные, так как старый reader уже был прочитан
	var buf3 bytes.Buffer
	zw3 := gzip.NewWriter(&buf3)
	_, err = zw3.Write([]byte("test data again"))
	if err != nil {
		t.Fatalf("Error writing to gzip second time: %v", err)
	}
	zw3.Close()

	zr2, err := pool.Get(&buf3)
	if err != nil {
		t.Fatalf("Error getting reader from pool second time: %v", err)
	}
	if zr2 != zr1 {
		t.Error("Expected same reader instance from pool")
	}
}

func TestCompressionManager(t *testing.T) {
	config := &CompressionConfig{
		EnableCompression: true,
		MinContentSize:    100,
	}
	manager := NewCompressionManager(config)

	// Тест ShouldCompress
	if !manager.ShouldCompress("application/json", 200) {
		t.Error("Expected compression for JSON content")
	}

	if manager.ShouldCompress("application/json", 50) {
		t.Error("Expected no compression for small content")
	}

	if manager.ShouldCompress("image/png", 200) {
		t.Error("Expected no compression for PNG content")
	}

	// Тест получения пулов
	if manager.GetWriterPool() == nil {
		t.Error("Expected non-nil writer pool")
	}

	if manager.GetReaderPool() == nil {
		t.Error("Expected non-nil reader pool")
	}

	// Тест IsCompressionEnabled
	if !manager.IsCompressionEnabled() {
		t.Error("Expected compression to be enabled")
	}
}

func TestCompressionConfig(t *testing.T) {
	// Тест с валидным уровнем сжатия
	config, err := ParseCompressionConfig()
	if err != nil {
		t.Fatalf("Error parsing config: %v", err)
	}

	if config.CompressionLevel < 1 || config.CompressionLevel > 9 {
		t.Errorf("Expected compression level between 1 and 9, got %d", config.CompressionLevel)
	}

	// Тест с невалидным уровнем сжатия (должен быть установлен по умолчанию)
	// Это потребует мокирования переменных окружения, что сложно в юнит-тестах
}

func BenchmarkCompressionWithPool(b *testing.B) {
	pool := NewWriterPool()
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		zw := pool.Get(buf)
		zw.Write(data)
		zw.Close()
		pool.Put(zw)
	}
}

func BenchmarkCompressionWithoutPool(b *testing.B) {
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		zw := gzip.NewWriter(buf)
		zw.Write(data)
		zw.Close()
	}
}

func TestOptimizedCompressWriter(t *testing.T) {
	config := &CompressionConfig{
		EnableCompression: true,
		MinContentSize:    10,
	}
	manager := NewCompressionManager(config)

	// Создаем тестовый ResponseWriter
	buf := &bytes.Buffer{}
	w := &testResponseWriter{
		header: make(http.Header),
		body:   buf,
	}

	cw := NewOptimizedCompressWriter(w, manager)

	// Тест WriteHeader
	cw.Header().Set("Content-Type", "application/json")
	cw.WriteHeader(200)

	if !cw.needCompress {
		t.Error("Expected compression to be enabled")
	}

	// Тест Write
	n, err := cw.Write([]byte("test data for compression"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n == 0 {
		t.Error("Expected bytes to be written")
	}

	// Тест Close
	err = cw.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
}

// testResponseWriter - тестовая реализация http.ResponseWriter
type testResponseWriter struct {
	header http.Header
	body   *bytes.Buffer
	status int
}

func (w *testResponseWriter) Header() http.Header {
	return w.header
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}
