package config

import (
	"flag"
	"fmt"
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

	paramAddress        HostAddress
	paramReportInterval int64
	paramPollInterval   int64
	paramRateLimit      int64
	paramKey            string
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
}

// Parse парсит переменные окружения и флаги командной строки.
//
// Заполняет конфигурацию значениями из переменных окружения и флагов.
// Приоритет отдается флагам командной строки.
// Должен вызываться после вызова метода Init().
func (ac *AgentConfig) Parse() {
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
	flag.Parse()

	// Address
	_, ok1 := problemVars["ADDRESS"]
	_, ok2 := problemVars["Address"]
	if !ok1 && !ok2 {
		ac.Address = ac.envs.Address
	} else {
		ac.Address = ac.paramAddress
	}

	// ReportInterval
	_, ok1 = problemVars["REPORT_INTERVAL"]
	_, ok2 = problemVars["ReportInterval"]
	if !ok1 && !ok2 {
		ac.ReportInterval = ac.envs.ReportInterval
	} else {
		ac.ReportInterval = ac.paramReportInterval
	}

	// PollInterval
	_, ok1 = problemVars["POLL_INTERVAL"]
	_, ok2 = problemVars["PollInterval"]
	if !ok1 && !ok2 {
		ac.PollInterval = ac.envs.PollInterval
	} else {
		ac.PollInterval = ac.paramPollInterval
	}

	// RateLimit
	_, ok1 = problemVars["RATE_LIMIT"]
	_, ok2 = problemVars["RateLimit"]
	if !ok1 && !ok2 {
		ac.RateLimit = ac.envs.RateLimit
	} else {
		ac.RateLimit = ac.paramRateLimit
	}

	// Key
	_, ok1 = problemVars["KEY"]
	_, ok2 = problemVars["Key"]
	if !ok1 && !ok2 {
		ac.Key = ac.envs.Key
	} else {
		ac.Key = ac.paramKey
	}
}
