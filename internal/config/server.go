package config

import (
	"flag"
	"fmt"
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
}

// Parse парсит переменные окружения и флаги командной строки.
//
// Заполняет конфигурацию значениями из переменных окружения и флагов.
// Приоритет отдается флагам командной строки.
// Должен вызываться после вызова метода Init().
func (se *ServerConfig) Parse() {

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
	flag.Parse()

	_, ok1 := problemVars["STORE_INTERVAL"]
	_, ok2 := problemVars["StoreInterval"]
	if !ok1 && !ok2 {
		se.StoreInterval = se.envs.StoreInterval
	} else {
		se.StoreInterval = se.paramStoreInterval
	}

	_, ok1 = problemVars["FILE_STORAGE_PATH"]
	_, ok2 = problemVars["FileStoragePath"]
	if !ok1 && !ok2 {
		se.FileStoragePath = se.envs.FileStoragePath
	} else {
		se.FileStoragePath = se.paramFileStoragePath
	}

	_, ok1 = problemVars["RESTORE"]
	_, ok2 = problemVars["Restore"]
	if !ok1 && !ok2 {
		se.Restore = se.envs.Restore
	} else {
		se.Restore = se.paramRestore
	}

	_, ok1 = problemVars["ADDRESS"]
	_, ok2 = problemVars["Address"]
	if !ok1 && !ok2 {
		se.Address = se.envs.Address
	} else {
		se.Address = se.paramAddress
	}

	_, ok1 = problemVars["DATABASE_DSN"]
	_, ok2 = problemVars["DatabaseDSN"]
	if !ok1 && !ok2 {
		se.DatabaseDSN = se.envs.DatabaseDSN
	} else {
		se.DatabaseDSN = se.paramDatabaseDSN
	}

	_, ok1 = problemVars["KEY"]
	_, ok2 = problemVars["Key"]
	if !ok1 && !ok2 {
		se.Key = se.envs.Key
	} else {
		se.Key = se.paramKey
	}

	_, ok1 = problemVars["AUDIT_FILE"]
	_, ok2 = problemVars["AuditFile"]
	if !ok1 && !ok2 {
		se.AuditFile = se.envs.AuditFile
	} else {
		se.AuditFile = se.paramAuditFile
	}

	_, ok1 = problemVars["AUDIT_URL"]
	_, ok2 = problemVars["AuditURL"]
	if !ok1 && !ok2 {
		se.AuditURL = se.envs.AuditURL
	} else {
		se.AuditURL = se.paramAuditURL
	}

	_, ok1 = problemVars["CRYPTO_KEY"]
	_, ok2 = problemVars["CryptoKey"]
	if !ok1 && !ok2 {
		se.CryptoKey = se.envs.CryptoKey
	} else {
		se.CryptoKey = se.paramCryptoKey
	}
}
