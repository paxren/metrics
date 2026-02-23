package config

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/caarlos0/env/v11"
)

// AgentConfigEnv представляет конфигурацию агента из переменных окружения.
//
// Используется для парсинга переменных окружения с тегами env.
type AgentConfigEnv struct {
	Address        HostAddress `env:"ADDRESS,notEmpty"`
	ReportInterval int64       `env:"REPORT_INTERVAL,notEmpty"`
	PollInterval   int64       `env:"POLL_INTERVAL,notEmpty"`
	RateLimit      int64       `env:"RATE_LIMIT,notEmpty"`
	Key            string      `env:"KEY,notEmpty"`
	CryptoKey      string      `env:"CRYPTO_KEY,notEmpty"`
}

// AgentConfig представляет полную конфигурацию агента.
//
// Объединяет параметры из переменных окружения и флагов командной строки.
// Приоритет отдается флагам командной строки.
type AgentConfig struct {
	envs           AgentConfigEnv
	Address        HostAddress
	ReportInterval int64
	PollInterval   int64
	RateLimit      int64
	Key            string
	CryptoKey      string

	paramAddress        HostAddress
	paramReportInterval int64
	paramPollInterval   int64
	paramRateLimit      int64
	paramKey            string
	paramCryptoKey      string
	paramConfigFile     string
}

// NewAgentConfig создаёт новую конфигурацию агента со значениями по умолчанию.
//
// Инициализирует параметры командной строки значениями по умолчанию.
//
// Возвращает:
//   - *AgentConfig: указатель на созданную конфигурацию
func NewAgentConfig() *AgentConfig {
	return &AgentConfig{
		paramAddress: *NewHostAddress(),
	}
}

// Init инициализирует флаги командной строки для конфигурации агента.
//
// Устанавливает флаги с их значениями по умолчанию и описаниями.
// Должен вызываться перед вызовом метода Parse().
func (ac *AgentConfig) Init() {
	flag.Var(&ac.paramAddress, "a", "Net address host:port")
	flag.Int64Var(&ac.paramReportInterval, "r", 10, "reportInterval")
	flag.Int64Var(&ac.paramPollInterval, "p", 2, "pollInterval")
	flag.Int64Var(&ac.paramRateLimit, "l", 1, "rateLimit")
	flag.StringVar(&ac.paramKey, "k", "", "hashKey")
	flag.StringVar(&ac.paramCryptoKey, "crypto-key", "", "path to public key file")
	flag.StringVar(&ac.paramConfigFile, "c", "", "path to config file")
	flag.StringVar(&ac.paramConfigFile, "config", "", "path to config file")
}

// Parse парсит переменные окружения и флаги командной строки.
//
// Заполняет конфигурацию значениями из переменных окружения и флагов.
// Приоритет отдается флагам командной строки.
// Должен вызываться после вызова метода Init().
func (ac *AgentConfig) Parse() {
	// 1. Сначала парсим переменную окружения CONFIG
	configPath := os.Getenv("CONFIG")

	// 2. Парсим флаги командной строки (включая -c/-config)
	flag.Parse()

	// 3. Определяем путь к конфигурационному файлу
	// Флаг имеет приоритет над переменной окружения
	if ac.paramConfigFile != "" {
		configPath = ac.paramConfigFile
	}

	// 4. Загружаем конфигурацию из файла, если указан
	var fileCfg *AgentConfigFile
	if configPath != "" {
		var err error
		fileCfg, err = LoadAgentConfigFile(configPath)
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
	err := env.ParseWithOptions(&ac.envs, env.Options{
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

	// Address
	if ac.paramAddress.Host != "" && ac.paramAddress.Port != 0 {
		ac.Address = ac.paramAddress
	} else if _, ok := problemVars["ADDRESS"]; !ok && ac.envs.Address.Host != "" {
		ac.Address = ac.envs.Address
	} else if fileCfg != nil && fileCfg.Address != "" {
		ha := NewHostAddress()
		if err := ha.Set(fileCfg.Address); err == nil {
			ac.Address = *ha
		}
	}

	// ReportInterval
	if ac.paramReportInterval != 0 {
		ac.ReportInterval = ac.paramReportInterval
	} else if _, ok := problemVars["REPORT_INTERVAL"]; !ok && ac.envs.ReportInterval != 0 {
		ac.ReportInterval = ac.envs.ReportInterval
	} else if fileCfg != nil && fileCfg.ReportInterval != "" {
		if val, err := ParseDuration(fileCfg.ReportInterval); err == nil {
			ac.ReportInterval = val
		}
	}

	// PollInterval
	if ac.paramPollInterval != 0 {
		ac.PollInterval = ac.paramPollInterval
	} else if _, ok := problemVars["POLL_INTERVAL"]; !ok && ac.envs.PollInterval != 0 {
		ac.PollInterval = ac.envs.PollInterval
	} else if fileCfg != nil && fileCfg.PollInterval != "" {
		if val, err := ParseDuration(fileCfg.PollInterval); err == nil {
			ac.PollInterval = val
		}
	}

	// RateLimit
	if ac.paramRateLimit != 0 {
		ac.RateLimit = ac.paramRateLimit
	} else if _, ok := problemVars["RATE_LIMIT"]; !ok && ac.envs.RateLimit != 0 {
		ac.RateLimit = ac.envs.RateLimit
	} else if fileCfg != nil && fileCfg.RateLimit != 0 {
		ac.RateLimit = fileCfg.RateLimit
	}

	// Key
	if ac.paramKey != "" {
		ac.Key = ac.paramKey
	} else if _, ok := problemVars["KEY"]; !ok && ac.envs.Key != "" {
		ac.Key = ac.envs.Key
	} else if fileCfg != nil && fileCfg.Key != "" {
		ac.Key = fileCfg.Key
	}

	// CryptoKey
	if ac.paramCryptoKey != "" {
		ac.CryptoKey = ac.paramCryptoKey
	} else if _, ok := problemVars["CRYPTO_KEY"]; !ok && ac.envs.CryptoKey != "" {
		ac.CryptoKey = ac.envs.CryptoKey
	} else if fileCfg != nil && fileCfg.CryptoKey != "" {
		ac.CryptoKey = fileCfg.CryptoKey
	}
}
