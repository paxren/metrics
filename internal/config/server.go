package config

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/caarlos0/env/v11"
)

// ServerConfigEnv представляет конфигурацию сервера из переменных окружения.
//
// Используется для парсинга переменных окружения с тегами env.
type ServerConfigEnv struct {
	StoreInterval   uint64      `env:"STORE_INTERVAL,notEmpty"`
	FileStoragePath string      `env:"FILE_STORAGE_PATH,notEmpty"`
	DatabaseDSN     string      `env:"DATABASE_DSN,notEmpty"`
	Restore         bool        `env:"RESTORE,notEmpty"`
	Address         HostAddress `env:"ADDRESS,notEmpty"`
	Key             string      `env:"KEY,notEmpty"`
	AuditFile       string      `env:"AUDIT_FILE,notEmpty"`
	AuditURL        string      `env:"AUDIT_URL,notEmpty"`
	CryptoKey       string      `env:"CRYPTO_KEY,notEmpty"`
}

// ServerConfig представляет полную конфигурацию сервера.
//
// Объединяет параметры из переменных окружения и флагов командной строки.
// Приоритет отдается флагам командной строки.
type ServerConfig struct {
	envs            ServerConfigEnv
	Address         HostAddress
	StoreInterval   uint64
	FileStoragePath string
	DatabaseDSN     string
	Key             string
	Restore         bool
	AuditFile       string
	AuditURL        string
	CryptoKey       string

	paramAddress         HostAddress
	paramStoreInterval   uint64
	paramFileStoragePath string
	paramDatabaseDSN     string
	paramKey             string
	paramRestore         bool
	paramAuditFile       string
	paramAuditURL        string
	paramCryptoKey       string
	paramConfigFile      string
}

// NewServerConfig создаёт новую конфигурацию сервера со значениями по умолчанию.
//
// Инициализирует параметры командной строки значениями по умолчанию.
//
// Возвращает:
//   - *ServerConfig: указатель на созданную конфигурацию
func NewServerConfig() *ServerConfig {

	return &ServerConfig{
		paramAddress: *NewHostAddress(),
	}

}

// Init инициализирует флаги командной строки для конфигурации сервера.
//
// Устанавливает флаги с их значениями по умолчанию и описаниями.
// Должен вызываться перед вызовом метода Parse().
func (se *ServerConfig) Init() {
	flag.Var(&se.paramAddress, "a", "Net address host:port")
	flag.Uint64Var(&se.paramStoreInterval, "i", 300, "storeInterval")
	flag.StringVar(&se.paramFileStoragePath, "f", "save_file", "fileStoragePath")
	flag.StringVar(&se.paramDatabaseDSN, "d", "", "fileStoragePath")
	flag.StringVar(&se.paramKey, "k", "", "key for hash")
	flag.BoolVar(&se.paramRestore, "r", false, "paramRestore")
	flag.StringVar(&se.paramAuditFile, "audit-file", "", "path to audit file")
	flag.StringVar(&se.paramAuditURL, "audit-url", "", "URL for audit logs")
	flag.StringVar(&se.paramCryptoKey, "crypto-key", "", "path to private key file")
	flag.StringVar(&se.paramConfigFile, "c", "", "path to config file")
	flag.StringVar(&se.paramConfigFile, "config", "", "path to config file")
}

// Parse парсит переменные окружения и флаги командной строки.
//
// Заполняет конфигурацию значениями из переменных окружения и флагов.
// Приоритет отдается флагам командной строки.
// Должен вызываться после вызова метода Init().
func (se *ServerConfig) Parse() {
	// 1. Сначала парсим переменную окружения CONFIG
	configPath := os.Getenv("CONFIG")

	// 2. Парсим флаги командной строки (включая -c/-config)
	flag.Parse()

	// 3. Определяем путь к конфигурационному файлу
	// Флаг имеет приоритет над переменной окружения
	if se.paramConfigFile != "" {
		configPath = se.paramConfigFile
	}

	// 4. Загружаем конфигурацию из файла, если указан
	var fileCfg *ServerConfigFile
	if configPath != "" {
		var err error
		fileCfg, err = LoadServerConfigFile(configPath)
		if err != nil {
			fmt.Printf("Warning: failed to load config file: %v\n", err)
		} else {
			// Валидация конфигурации из файла
			if err := fileCfg.Validate(); err != nil {
				fmt.Printf("Warning: invalid config file: %v\n", err)
				fileCfg = nil
			}
		}
	}

	// 5. Парсим переменные окружения
	err := env.ParseWithOptions(&se.envs, env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(HostAddress{}): func(v string) (interface{}, error) {
				ha := NewHostAddress()
				err := ha.Set(v)
				return *ha, err
			},
		},
	})

	problemVars := make(map[string]bool)

	if err != nil {
		if err, ok := err.(env.AggregateError); ok {
			for _, v := range err.Errors {
				fmt.Printf("err.Error: %T\n", v)
				fmt.Printf("err.Error: %v\n", v)

				if err1, ok := v.(env.EmptyVarError); ok {
					problemVars[err1.Key] = true
				}

				if err2, ok := v.(env.ParseError); ok {
					problemVars[err2.Name] = true
				}

				if _, ok := v.(HostAddressParseError); ok {
					problemVars["ADDRESS"] = true
				}
			}
		}
	}

	fmt.Printf("problemVars = %v", problemVars)

	// 6. Применяем значения с учетом приоритета:
	//    Флаги > Переменные окружения > Файл конфигурации

	// StoreInterval
	if se.paramStoreInterval != 0 {
		se.StoreInterval = se.paramStoreInterval
	} else if _, ok := problemVars["STORE_INTERVAL"]; !ok && se.envs.StoreInterval != 0 {
		se.StoreInterval = se.envs.StoreInterval
	} else if fileCfg != nil && fileCfg.StoreInterval != "" {
		if val, err := ParseDurationUint(fileCfg.StoreInterval); err == nil {
			se.StoreInterval = val
		}
	}

	// FileStoragePath
	if se.paramFileStoragePath != "" {
		se.FileStoragePath = se.paramFileStoragePath
	} else if _, ok := problemVars["FILE_STORAGE_PATH"]; !ok && se.envs.FileStoragePath != "" {
		se.FileStoragePath = se.envs.FileStoragePath
	} else if fileCfg != nil && fileCfg.StoreFile != "" {
		se.FileStoragePath = fileCfg.StoreFile
	}

	// Restore
	if se.paramRestore {
		se.Restore = se.paramRestore
	} else if _, ok := problemVars["RESTORE"]; !ok {
		se.Restore = se.envs.Restore
	} else if fileCfg != nil {
		se.Restore = fileCfg.Restore
	}

	// Address
	if se.paramAddress.Host != "" && se.paramAddress.Port != 0 {
		se.Address = se.paramAddress
	} else if _, ok := problemVars["ADDRESS"]; !ok && se.envs.Address.Host != "" {
		se.Address = se.envs.Address
	} else if fileCfg != nil && fileCfg.Address != "" {
		ha := NewHostAddress()
		if err := ha.Set(fileCfg.Address); err == nil {
			se.Address = *ha
		}
	}

	// DatabaseDSN
	if se.paramDatabaseDSN != "" {
		se.DatabaseDSN = se.paramDatabaseDSN
	} else if _, ok := problemVars["DATABASE_DSN"]; !ok && se.envs.DatabaseDSN != "" {
		se.DatabaseDSN = se.envs.DatabaseDSN
	} else if fileCfg != nil && fileCfg.DatabaseDSN != "" {
		se.DatabaseDSN = fileCfg.DatabaseDSN
	}

	// Key
	if se.paramKey != "" {
		se.Key = se.paramKey
	} else if _, ok := problemVars["KEY"]; !ok && se.envs.Key != "" {
		se.Key = se.envs.Key
	} else if fileCfg != nil && fileCfg.Key != "" {
		se.Key = fileCfg.Key
	}

	// AuditFile
	if se.paramAuditFile != "" {
		se.AuditFile = se.paramAuditFile
	} else if _, ok := problemVars["AUDIT_FILE"]; !ok && se.envs.AuditFile != "" {
		se.AuditFile = se.envs.AuditFile
	} else if fileCfg != nil && fileCfg.AuditFile != "" {
		se.AuditFile = fileCfg.AuditFile
	}

	// AuditURL
	if se.paramAuditURL != "" {
		se.AuditURL = se.paramAuditURL
	} else if _, ok := problemVars["AUDIT_URL"]; !ok && se.envs.AuditURL != "" {
		se.AuditURL = se.envs.AuditURL
	} else if fileCfg != nil && fileCfg.AuditURL != "" {
		se.AuditURL = fileCfg.AuditURL
	}

	// CryptoKey
	if se.paramCryptoKey != "" {
		se.CryptoKey = se.paramCryptoKey
	} else if _, ok := problemVars["CRYPTO_KEY"]; !ok && se.envs.CryptoKey != "" {
		se.CryptoKey = se.envs.CryptoKey
	} else if fileCfg != nil && fileCfg.CryptoKey != "" {
		se.CryptoKey = fileCfg.CryptoKey
	}
}
