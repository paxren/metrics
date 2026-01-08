package handler

import (
	"github.com/caarlos0/env/v11"
)

// CompressionConfig содержит настройки для системы сжатия
type CompressionConfig struct {
	// WriterPoolSize максимальный размер пула писателей (экспериментальный)
	WriterPoolSize int `env:"COMPRESSION_WRITER_POOL_SIZE" envDefault:"100"`

	// ReaderPoolSize максимальный размер пула читателей (экспериментальный)
	ReaderPoolSize int `env:"COMPRESSION_READER_POOL_SIZE" envDefault:"50"`

	// CompressionLevel уровень сжатия gzip (1-9, 6 по умолчанию)
	CompressionLevel int `env:"COMPRESSION_LEVEL" envDefault:"6"`

	// EnableCompression включение/выключение сжатия
	EnableCompression bool `env:"ENABLE_COMPRESSION" envDefault:"true"`

	// MinContentSize минимальный размер контента для сжатия в байтах
	MinContentSize int `env:"COMPRESSION_MIN_CONTENT_SIZE" envDefault:"1024"`
}

// ParseCompressionConfig парсит конфигурацию из переменных окружения
func ParseCompressionConfig() (*CompressionConfig, error) {
	cfg := &CompressionConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	// Валидация уровня сжатия
	if cfg.CompressionLevel < 1 || cfg.CompressionLevel > 9 {
		cfg.CompressionLevel = 6
	}

	return cfg, nil
}
