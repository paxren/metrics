package main

import (
	"fmt"

	"github.com/paxren/metrics/internal/agent"
	"github.com/paxren/metrics/internal/config"
)

var (
	agentConfig = config.NewAgentConfig()
)

func init() {
	agentConfig.Init()
}

func main() {
	// Парсим переменные окружения и флаги
	agentConfig.Parse()

	// Выводим конфигурацию для отладки
	fmt.Printf("Address: %v\n", agentConfig.Address)
	fmt.Printf("Report interval: %d\n", agentConfig.ReportInterval)
	fmt.Printf("Poll interval: %d\n", agentConfig.PollInterval)
	fmt.Printf("Rate limiter: %d\n", agentConfig.RateLimit)
	fmt.Printf("Key: %s\n", agentConfig.Key)

	// Создаем и запускаем агента с новой конфигурацией
	agentInstance := agent.NewAgentExtended(
		agentConfig.Address,
		agentConfig.Key,
		agentConfig.RateLimit,
		agentConfig.PollInterval,
		agentConfig.ReportInterval,
	)

	done := agentInstance.Start()
	<-done
}
