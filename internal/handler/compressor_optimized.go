package handler

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// OptimizedCompressWriter реализует http.ResponseWriter с поддержкой пулов
type OptimizedCompressWriter struct {
	w               http.ResponseWriter
	zw              *gzip.Writer
	pool            *WriterPool
	manager         *CompressionManager
	needCompress    bool
	needCompressSet bool
	mu              sync.Mutex // Защита от гонок
}

// NewOptimizedCompressWriter создает новый оптимизированный компрессор
func NewOptimizedCompressWriter(w http.ResponseWriter, manager *CompressionManager) *OptimizedCompressWriter {
	return &OptimizedCompressWriter{
		w:               w,
		zw:              nil,
		pool:            manager.GetWriterPool(),
		manager:         manager,
		needCompress:    false,
		needCompressSet: false,
	}
}

// Header возвращает заголовки ответа
func (c *OptimizedCompressWriter) Header() http.Header {
	return c.w.Header()
}

// Write записывает данные с сжатием если необходимо
func (c *OptimizedCompressWriter) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.needCompressSet {
		fmt.Println("ERROR: needCompress not set before Write")
	}

	if c.needCompress {
		if c.zw == nil {
			c.zw = c.pool.Get(c.w)
		}
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

// WriteHeader устанавливает заголовки и определяет нужно ли сжатие
func (c *OptimizedCompressWriter) WriteHeader(statusCode int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	contentType := c.w.Header().Get("Content-Type")
	contentLength := 0
	if lengthStr := c.w.Header().Get("Content-Length"); lengthStr != "" {
		fmt.Sscanf(lengthStr, "%d", &contentLength)
	}

	c.needCompress = statusCode < 300 &&
		c.manager.ShouldCompress(contentType, contentLength)
	c.needCompressSet = true

	if c.needCompress {
		c.w.Header().Set("Content-Encoding", "gzip")
		c.w.Header().Del("Content-Length") // Длина изменится после сжатия
	}

	fmt.Printf("Compression enabled: %v, ContentType: %s, StatusCode: %d\n",
		c.needCompress, contentType, statusCode)

	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и возвращает его в пул
func (c *OptimizedCompressWriter) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.zw != nil {
		err := c.zw.Close()
		c.pool.Put(c.zw)
		c.zw = nil
		return err
	}
	return nil
}

// OptimizedCompressReader реализует io.ReadCloser с поддержкой пулов
type OptimizedCompressReader struct {
	r      io.ReadCloser
	zr     *gzip.Reader
	pool   *ReaderPool
	closed bool
	mu     sync.Mutex
}

// NewOptimizedCompressReader создает новый оптимизированный читатель
func NewOptimizedCompressReader(r io.ReadCloser, pool *ReaderPool) (*OptimizedCompressReader, error) {
	zr, err := pool.Get(r)
	if err != nil {
		return nil, err
	}

	return &OptimizedCompressReader{
		r:      r,
		zr:     zr,
		pool:   pool,
		closed: false,
	}, nil
}

// Read читает данные с декомпрессией
func (c *OptimizedCompressReader) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.EOF
	}

	return c.zr.Read(p)
}

// Close закрывает читатели и возвращает gzip.Reader в пул
func (c *OptimizedCompressReader) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Возвращаем gzip.Reader в пул
	if c.zr != nil {
		c.pool.Put(c.zr)
		c.zr = nil
	}

	// Закрываем исходный reader
	return c.r.Close()
}

// Глобальный менеджер сжатия
var globalCompressionManager *CompressionManager
var compressionManagerOnce sync.Once

// GetCompressionManager возвращает глобальный менеджер сжатия (singleton)
func GetCompressionManager() *CompressionManager {
	compressionManagerOnce.Do(func() {
		config, err := ParseCompressionConfig()
		if err != nil {
			// Используем конфигурацию по умолчанию при ошибке
			config = &CompressionConfig{
				EnableCompression: true,
				CompressionLevel:  6,
				MinContentSize:    1024,
			}
		}
		globalCompressionManager = NewCompressionManager(config)
	})
	return globalCompressionManager
}

// OptimizedGzipMiddleware middleware с поддержкой пулов
func OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manager := GetCompressionManager()

		if !manager.IsCompressionEnabled() {
			h.ServeHTTP(w, r)
			return
		}

		ow := w

		// Проверяем поддержку gzip клиентом
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		if supportsGzip {
			cw := NewOptimizedCompressWriter(w, manager)
			ow = cw
			defer cw.Close()
		}

		// Проверяем сжатие тела запроса
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")

		if sendsGzip {
			cr, err := NewOptimizedCompressReader(r.Body, manager.GetReaderPool())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	}
}
