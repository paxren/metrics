package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paxren/metrics/internal/models"
)

func TestMakeSavedRepo(t *testing.T) {
	repo := MakeMemStorage()
	fileName := "test_file.json"

	// Тест создания без тикера (interval = 0)
	fs := MakeSavedRepo(repo, fileName, 0)
	if fs == nil {
		t.Error("MakeSavedRepo returned nil")
	}
	if fs.fileName != fileName {
		t.Errorf("Expected fileName %s, got %s", fileName, fs.fileName)
	}
	if fs.ticker != nil {
		t.Error("Expected ticker to be nil for interval 0")
	}

	// Тест создания с тикером
	fsWithTicker := MakeSavedRepo(repo, fileName, 1)
	if fsWithTicker == nil {
		t.Error("MakeSavedRepo returned nil for ticker case")
	}
	if fsWithTicker.fileName != fileName {
		t.Errorf("Expected fileName %s, got %s", fileName, fsWithTicker.fileName)
	}
	if fsWithTicker.ticker == nil {
		t.Error("Expected ticker to be non-nil for interval > 0")
	}
	// Проверяем, что канал тикера не nil (интервал проверить сложнее)
	if fsWithTicker.ticker.C == nil {
		t.Error("Expected ticker channel to be non-nil")
	}
}

func TestFileSaver_UpdateGauge(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	fs := MakeSavedRepo(repo, fileName, 0) // Без тикера

	// Тест успешного обновления gauge
	err := fs.UpdateGauge("test_gauge", 123.45)
	if err != nil {
		t.Errorf("UpdateGauge returned error: %v", err)
	}

	// Проверяем, что метрика сохранилась в репозитории
	value, err := repo.GetGauge("test_gauge")
	if err != nil {
		t.Errorf("GetGauge returned error: %v", err)
	}
	if value != 123.45 {
		t.Errorf("Expected gauge value 123.45, got %f", value)
	}

	// Проверяем, что файл создался и содержит данные
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Error("File was not created")
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	dataStr := string(data)
	if !contains(dataStr, "test_gauge") {
		t.Error("File does not contain test_gauge")
	}
	if !contains(dataStr, "123.45") {
		t.Error("File does not contain gauge value 123.45")
	}
}

func TestFileSaver_UpdateCounter(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	fs := MakeSavedRepo(repo, fileName, 0) // Без тикера

	// Тест успешного обновления counter
	err := fs.UpdateCounter("test_counter", 100)
	if err != nil {
		t.Errorf("UpdateCounter returned error: %v", err)
	}

	// Проверяем, что метрика сохранилась в репозитории
	value, err := repo.GetCounter("test_counter")
	if err != nil {
		t.Errorf("GetCounter returned error: %v", err)
	}
	if value != 100 {
		t.Errorf("Expected counter value 100, got %d", value)
	}

	// Проверяем, что файл создался и содержит данные
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Error("File was not created")
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	dataStr := string(data)
	if !contains(dataStr, "test_counter") {
		t.Error("File does not contain test_counter")
	}
	if !contains(dataStr, "100") {
		t.Error("File does not contain counter value 100")
	}
}

func TestFileSaver_Save(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Добавляем тестовые данные
	repo.UpdateGauge("gauge1", 111.11)
	repo.UpdateGauge("gauge2", 222.22)
	repo.UpdateCounter("counter1", 50)
	repo.UpdateCounter("counter2", 75)

	fs := MakeSavedRepo(repo, fileName, 0)

	// Тест сохранения
	err := fs.Save()
	if err != nil {
		t.Errorf("Save returned error: %v", err)
	}

	// Проверяем, что файл создался
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Error("File was not created")
	}

	// Проверяем содержимое файла
	data, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}

	dataStr := string(data)
	// Проверяем, что JSON содержит все метрики
	if !contains(dataStr, "gauge1") {
		t.Error("File does not contain gauge1")
	}
	if !contains(dataStr, "gauge2") {
		t.Error("File does not contain gauge2")
	}
	if !contains(dataStr, "counter1") {
		t.Error("File does not contain counter1")
	}
	if !contains(dataStr, "counter2") {
		t.Error("File does not contain counter2")
	}
	if !contains(dataStr, "111.11") {
		t.Error("File does not contain gauge1 value")
	}
	if !contains(dataStr, "222.22") {
		t.Error("File does not contain gauge2 value")
	}
	if !contains(dataStr, "50") {
		t.Error("File does not contain counter1 value")
	}
	if !contains(dataStr, "75") {
		t.Error("File does not contain counter2 value")
	}
}

func TestFileSaver_Load(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Создаем JSON данные для загрузки
	metrics := []models.Metrics{
		{
			ID:    "loaded_gauge",
			MType: models.Gauge,
			Value: func() *float64 { v := 333.33; return &v }(),
		},
		{
			ID:    "loaded_counter",
			MType: models.Counter,
			Delta: func() *int64 { v := int64(200); return &v }(),
		},
	}

	data, err := json.MarshalIndent(metrics, "", "\t")
	if err != nil {
		t.Errorf("Failed to marshal JSON: %v", err)
	}
	err = os.WriteFile(fileName, data, 0644)
	if err != nil {
		t.Errorf("Failed to write file: %v", err)
	}

	fs := MakeSavedRepo(repo, fileName, 0)

	// Тест загрузки
	err = fs.Load(fileName)
	if err != nil {
		t.Errorf("Load returned error: %v", err)
	}

	// Проверяем, что метрики загрузились в репозиторий
	gaugeValue, err := repo.GetGauge("loaded_gauge")
	if err != nil {
		t.Errorf("GetGauge returned error: %v", err)
	}
	if gaugeValue != 333.33 {
		t.Errorf("Expected gauge value 333.33, got %f", gaugeValue)
	}

	counterValue, err := repo.GetCounter("loaded_counter")
	if err != nil {
		t.Errorf("GetCounter returned error: %v", err)
	}
	if counterValue != 200 {
		t.Errorf("Expected counter value 200, got %d", counterValue)
	}
}

func TestFileSaver_Load_InvalidJSON(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Записываем некорректный JSON
	invalidJSON := `{"invalid": json}`
	err := os.WriteFile(fileName, []byte(invalidJSON), 0644)
	if err != nil {
		t.Errorf("Failed to write file: %v", err)
	}

	fs := MakeSavedRepo(repo, fileName, 0)

	// Тест загрузки некорректного JSON
	err = fs.Load(fileName)
	if err == nil {
		t.Error("Load should return error for invalid JSON")
	}
}

func TestFileSaver_Load_UnknownMetricType(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Создаем метрику с неизвестным типом
	metrics := []models.Metrics{
		{
			ID:    "unknown_metric",
			MType: "unknown_type",
		},
	}

	data, err := json.MarshalIndent(metrics, "", "\t")
	if err != nil {
		t.Errorf("Failed to marshal JSON: %v", err)
	}
	err = os.WriteFile(fileName, data, 0644)
	if err != nil {
		t.Errorf("Failed to write file: %v", err)
	}

	fs := MakeSavedRepo(repo, fileName, 0)

	// Тест загрузки метрики с неизвестным типом
	err = fs.Load(fileName)
	if err == nil {
		t.Error("Load should return error for unknown metric type")
	}
	if err != nil && !contains(err.Error(), "неизвестный тип метрики") {
		t.Errorf("Expected error message to contain 'неизвестный тип метрики', got: %v", err)
	}
}

func TestFileSaver_Load_NonExistentFile(t *testing.T) {
	repo := MakeMemStorage()
	nonExistentFile := "non_existent_file.json"

	fs := MakeSavedRepo(repo, nonExistentFile, 0)

	// Тест загрузки несуществующего файла
	err := fs.Load(nonExistentFile)
	if err == nil {
		t.Error("Load should return error for non-existent file")
	}
}

func TestFileSaver_Save_Error(t *testing.T) {
	repo := MakeMemStorage()
	// Используем недопустимое имя файла для создания ошибки
	invalidFileName := "/invalid/path/file.json"

	repo.UpdateGauge("test_gauge", 123.45)

	fs := MakeSavedRepo(repo, invalidFileName, 0)

	// Тест ошибки сохранения
	err := fs.Save()
	if err == nil {
		t.Error("Save should return error for invalid file path")
	}
}

func TestFileSaver_UpdateGauge_WithTicker(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Создаем FileSaver с тикером
	fs := MakeSavedRepo(repo, fileName, 1) // 1 секунда

	// Обновляем gauge - должно сработать тикер
	err := fs.UpdateGauge("ticker_gauge", 999.99)
	if err != nil {
		t.Errorf("UpdateGauge returned error: %v", err)
	}

	// Ждем немного, чтобы тикер сработал
	time.Sleep(1500 * time.Millisecond)

	// Проверяем, что файл обновился через тикер
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Error("File was not created by ticker")
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	dataStr := string(data)
	if !contains(dataStr, "ticker_gauge") {
		t.Error("File does not contain ticker_gauge")
	}
	if !contains(dataStr, "999.99") {
		t.Error("File does not contain ticker gauge value")
	}
}

func TestFileSaver_UpdateCounter_WithTicker(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Создаем FileSaver с тикером
	fs := MakeSavedRepo(repo, fileName, 1) // 1 секунда

	// Обновляем counter - должно сработать тикер
	err := fs.UpdateCounter("ticker_counter", 888)
	if err != nil {
		t.Errorf("UpdateCounter returned error: %v", err)
	}

	// Ждем немного, чтобы тикер сработал
	time.Sleep(1500 * time.Millisecond)

	// Проверяем, что файл обновился через тикер
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Error("File was not created by ticker")
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	dataStr := string(data)
	if !contains(dataStr, "ticker_counter") {
		t.Error("File does not contain ticker_counter")
	}
	if !contains(dataStr, "888") {
		t.Error("File does not contain ticker counter value")
	}
}

// Вспомогательные функции

func createTempFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	return filepath.Join(tmpDir, "test_metrics.json")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Тест для проверки работы с пустым репозиторием
func TestFileSaver_Save_EmptyRepository(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	fs := MakeSavedRepo(repo, fileName, 0)

	// Сохраняем пустой репозиторий
	err := fs.Save()
	if err != nil {
		t.Errorf("Save returned error: %v", err)
	}

	// Проверяем, что создался файл с пустым массивом
	data, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	// Проверяем, что файл содержит пустой JSON массив (может быть с переносом строки или без)
	dataStr := string(data)
	if dataStr != "[]" && dataStr != "[]\n" {
		t.Errorf("Expected empty array, got: %s", dataStr)
	}
}

// Тест для проверки загрузки пустого файла
func TestFileSaver_Load_EmptyFile(t *testing.T) {
	repo := MakeMemStorage()
	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Создаем пустой файл
	err := os.WriteFile(fileName, []byte("[]"), 0644)
	if err != nil {
		t.Errorf("Failed to write file: %v", err)
	}

	fs := MakeSavedRepo(repo, fileName, 0)

	// Загружаем пустой файл
	err = fs.Load(fileName)
	if err != nil {
		t.Errorf("Load returned error: %v", err)
	}

	// Проверяем, что репозиторий остался пустым
	gauges := repo.GetGaugesKeys()
	counters := repo.GetCountersKeys()
	if len(gauges) != 0 {
		t.Errorf("Expected empty gauges, got %d", len(gauges))
	}
	if len(counters) != 0 {
		t.Errorf("Expected empty counters, got %d", len(counters))
	}
}

// MockRepository для тестирования ошибок
type MockRepository struct {
	gauges   map[string]float64
	counters map[string]int64
	errors   map[string]bool
}

func (m *MockRepository) UpdateGauge(key string, value float64) error {
	if m.errors[key] {
		return fmt.Errorf("mock error for gauge %s", key)
	}
	m.gauges[key] = value
	return nil
}

func (m *MockRepository) UpdateCounter(key string, value int64) error {
	if m.errors[key] {
		return fmt.Errorf("mock error for counter %s", key)
	}
	m.counters[key] += value
	return nil
}

func (m *MockRepository) GetGauge(key string) (float64, error) {
	if m.errors[key] {
		return 0, fmt.Errorf("mock error for gauge %s", key)
	}
	val, ok := m.gauges[key]
	if !ok {
		return 0, ErrGaugeNotFound
	}
	return val, nil
}

func (m *MockRepository) GetCounter(key string) (int64, error) {
	if m.errors[key] {
		return 0, fmt.Errorf("mock error for counter %s", key)
	}
	val, ok := m.counters[key]
	if !ok {
		return 0, ErrCounterNotFound
	}
	return val, nil
}

func (m *MockRepository) GetGaugesKeys() []string {
	keys := make([]string, 0, len(m.gauges))
	for k := range m.gauges {
		keys = append(keys, k)
	}
	return keys
}

func (m *MockRepository) GetCountersKeys() []string {
	keys := make([]string, 0, len(m.counters))
	for k := range m.counters {
		keys = append(keys, k)
	}
	return keys
}

// Тест для проверки ошибок при получении метрик из репозитория
func TestFileSaver_Save_RepositoryError(t *testing.T) {
	// Создаем mock репозиторий, который возвращает ошибки
	mockRepo := &MockRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		errors:   make(map[string]bool),
	}

	fileName := createTempFile(t)
	defer os.Remove(fileName)

	// Добавляем метрики и устанавливаем ошибки для них
	mockRepo.gauges["error_gauge"] = 123.45
	mockRepo.counters["error_counter"] = 100
	mockRepo.errors["error_gauge"] = true
	mockRepo.errors["error_counter"] = true

	fs := MakeSavedRepo(mockRepo, fileName, 0)

	// Тест ошибки при сохранении с ошибками в репозитории
	err := fs.Save()
	if err == nil {
		t.Error("Save should return error when repository GetGauge fails")
	}
	if err != nil && !contains(err.Error(), "mock error") {
		t.Errorf("Expected error message to contain 'mock error', got: %v", err)
	}
}
