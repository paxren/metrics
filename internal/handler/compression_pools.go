package handler

import (
	"compress/gzip"
	"io"
	"sync"
)

// WriterPool управляет пулом gzip.Writer для переиспользования
type WriterPool struct {
	pool sync.Pool
}

// NewWriterPool создает новый пул для gzip.Writer
func NewWriterPool() *WriterPool {
	return &WriterPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Создаем writer с io.Discard как заглушку
				// Реальный writer будет установлен через Reset()
				return gzip.NewWriter(io.Discard)
			},
		},
	}
}

// Get получает gzip.Writer из пула и настраивает его для работы с w
func (p *WriterPool) Get(w io.Writer) *gzip.Writer {
	zw := p.pool.Get().(*gzip.Writer)
	zw.Reset(w)
	return zw
}

// Put возвращает gzip.Writer в пул после использования
func (p *WriterPool) Put(zw *gzip.Writer) {
	if zw != nil {
		zw.Close() // Закрываем для сброса буферов
		p.pool.Put(zw)
	}
}

// ReaderPool управляет пулом gzip.Reader для переиспользования
type ReaderPool struct {
	pool sync.Pool
}

// NewReaderPool создает новый пул для gzip.Reader
func NewReaderPool() *ReaderPool {
	return &ReaderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(gzip.Reader)
			},
		},
	}
}

// Get получает gzip.Reader из пула и настраивает его для работы с r
func (p *ReaderPool) Get(r io.Reader) (*gzip.Reader, error) {
	zr := p.pool.Get().(*gzip.Reader)
	err := zr.Reset(r)
	if err != nil {
		// При ошибке возвращаем в пул и возвращаем ошибку
		p.pool.Put(zr)
		return nil, err
	}
	return zr, nil
}

// Put возвращает gzip.Reader в пул после использования
func (p *ReaderPool) Put(zr *gzip.Reader) {
	if zr != nil {
		zr.Close() // Закрываем для очистки
		p.pool.Put(zr)
	}
}

// CompressionManager управляет всеми пулами и конфигурацией
type CompressionManager struct {
	writerPool *WriterPool
	readerPool *ReaderPool
	config     *CompressionConfig
}

// NewCompressionManager создает новый менеджер сжатия
func NewCompressionManager(config *CompressionConfig) *CompressionManager {
	return &CompressionManager{
		writerPool: NewWriterPool(),
		readerPool: NewReaderPool(),
		config:     config,
	}
}

// GetWriterPool возвращает пул писателей
func (cm *CompressionManager) GetWriterPool() *WriterPool {
	return cm.writerPool
}

// GetReaderPool возвращает пул читателей
func (cm *CompressionManager) GetReaderPool() *ReaderPool {
	return cm.readerPool
}

// GetConfig возвращает конфигурацию
func (cm *CompressionManager) GetConfig() *CompressionConfig {
	return cm.config
}

// IsCompressionEnabled проверяет включено ли сжатие
func (cm *CompressionManager) IsCompressionEnabled() bool {
	return cm.config.EnableCompression
}

// ShouldCompress проверяет, нужно ли сжимать контент
func (cm *CompressionManager) ShouldCompress(contentType string, contentLength int) bool {
	if !cm.config.EnableCompression {
		return false
	}

	if contentLength > 0 && contentLength < cm.config.MinContentSize {
		return false
	}

	// Проверяем тип контента
	return contentType == "application/json" ||
		contentType == "text/html" ||
		contentType == "text/plain" ||
		contentType == "text/css" ||
		contentType == "application/javascript"
}
