package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/paxren/metrics/internal/agent"
	"github.com/paxren/metrics/internal/config"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
	agentConfig  = config.NewAgentConfig()
)

func init() {
	agentConfig.Init()
}

func main() {
	// Выводим информацию о сборке
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	// Обработка SIGTERM и SIGINT для graceful shutdown
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Парсим переменные окружения и флаги
	agentConfig.Parse()

	// Выводим конфигурацию для отладки
	fmt.Printf("Address: %v\n", agentConfig.Address)
	fmt.Printf("Report interval: %d\n", agentConfig.ReportInterval)
	fmt.Printf("Poll interval: %d\n", agentConfig.PollInterval)
	fmt.Printf("Rate limiter: %d\n", agentConfig.RateLimit)
	fmt.Printf("Key: %s\n", agentConfig.Key)
	fmt.Printf("Crypto key: %s\n", agentConfig.CryptoKey)

	// Создаем и запускаем агента с новой конфигурацией
	agentInstance := agent.NewAgentExtended(
		agentConfig.Address,
		agentConfig.Key,
		agentConfig.RateLimit,
		agentConfig.PollInterval,
		agentConfig.ReportInterval,
		agentConfig.CryptoKey,
	)

	done := agentInstance.Start()

	// Ожидаем либо сигнал завершения от ОС, либо завершение работы агента
	select {
	case <-rootCtx.Done():
		// Получен сигнал SIGINT или SIGTERM
		stop()
		agentInstance.Finish()
	case <-done:
		// Агент завершил работу самостоятельно
		stop()
	}
}
