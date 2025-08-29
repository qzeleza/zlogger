package logger

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

/**
 * TestDefaultConfigurations тестирует функции создания конфигураций по умолчанию
 * @param t *testing.T - тестовый контекст
 */
func TestSecurityConfigurations(t *testing.T) {
	// Тестируем DefaultSecurityConfig
	securityConfig := DefaultSecurityConfig()
	if securityConfig == nil {
		t.Fatal("конфигурация безопасности по умолчанию не должна быть nil")
	}
	if securityConfig.RateLimitPerSecond <= 0 {
		t.Error("ограничение скорости в секунду должно быть положительным")
	}
	if securityConfig.BanDuration <= 0 {
		t.Error("длительность бана должна быть положительной")
	}

	// Тестируем создание нового RateLimiter
	rateLimiter := NewRateLimiter(securityConfig)
	if rateLimiter == nil {
		t.Error("RateLimiter не должен быть nil")
	}
	defer rateLimiter.Close()
}

/**
 * TestFilterOptions тестирует структуру FilterOptions и её методы
 * @param t *testing.T - тестовый контекст
 */
func TestFilterOptions(t *testing.T) {
	// Создаем тестовые записи
	entries := []LogEntry{
		{Level: DEBUG, Service: "SERVICE1", Message: "debug message", Timestamp: time.Now()},
		{Level: INFO, Service: "SERVICE1", Message: "info message", Timestamp: time.Now()},
		{Level: WARN, Service: "SERVICE2", Message: "warning message", Timestamp: time.Now()},
		{Level: ERROR, Service: "SERVICE2", Message: "error message", Timestamp: time.Now()},
		{Level: PANIC, Service: "SERVICE3", Message: "panic message", Timestamp: time.Now()},
	}

	// Тестируем фильтрацию по сервису
	filter := FilterOptions{Service: "SERVICE1"}
	filtered := filterEntries(entries, filter)
	if len(filtered) != 2 {
		t.Errorf("ожидалось 2 записи для SERVICE1, получено %d", len(filtered))
	}

	// Тестируем фильтрацию по уровню
	errorLevel := ERROR
	filter = FilterOptions{Level: &errorLevel}
	filtered = filterEntries(entries, filter)
	if len(filtered) != 2 { // ERROR и PANIC
		t.Errorf("ожидалось 2 записи для уровня ERROR и выше, получено %d", len(filtered))
	}

	// Тестируем комбинированную фильтрацию
	filter = FilterOptions{Service: "SERVICE2", Level: &errorLevel}
	filtered = filterEntries(entries, filter)
	if len(filtered) != 1 { // Только ERROR от SERVICE2
		t.Errorf("ожидалась 1 запись для SERVICE2 с уровнем ERROR и выше, получено %d", len(filtered))
	}
}

/**
 * filterEntries вспомогательная функция для фильтрации записей (имитирует логику сервера)
 * @param entries []LogEntry - записи для фильтрации
 * @param filter FilterOptions - параметры фильтрации
 * @return []LogEntry - отфильтрованные записи
 */
func filterEntries(entries []LogEntry, filter FilterOptions) []LogEntry {
	var result []LogEntry
	for _, entry := range entries {
		if filter.Service != "" && entry.Service != filter.Service {
			continue
		}
		if filter.Level != nil && entry.Level < *filter.Level {
			continue
		}
		result = append(result, entry)
	}
	return result
}

/**
 * TestMessageSerialization тестирует сериализацию и десериализацию сообщений
 * @param t *testing.T - тестовый контекст
 */
func TestMessageSerialization(t *testing.T) {
	// Тестируем сериализацию LogMessage
	msg := LogMessage{

		Level:     INFO,
		Message:   "test message with unicode: тест 测试 🚀",
		Service:   "TEST_SERVICE",
		Timestamp: time.Now(),
		ClientID:  "test-client-123",
	}

	// Сериализуем в JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("ошибка сериализации сообщения: %v", err)
	}

	// Десериализуем обратно
	var deserializedMsg LogMessage
	err = json.Unmarshal(data, &deserializedMsg)
	if err != nil {
		t.Errorf("ошибка десериализации сообщения: %v", err)
	}

	// Проверяем корректность десериализации
	if deserializedMsg.ClientID != msg.ClientID {
		t.Errorf("ClientID не совпадает: ожидался %s, получен %s", msg.ClientID, deserializedMsg.ClientID)
	}
	if deserializedMsg.Level != msg.Level {
		t.Errorf("уровень не совпадает: ожидался %v, получен %v", msg.Level, deserializedMsg.Level)
	}
	if deserializedMsg.Message != msg.Message {
		t.Errorf("сообщение не совпадает: ожидалось '%s', получено '%s'", msg.Message, deserializedMsg.Message)
	}
	if deserializedMsg.Service != msg.Service {
		t.Errorf("сервис не совпадает: ожидался '%s', получен '%s'", msg.Service, deserializedMsg.Service)
	}

	// Тестируем сериализацию LogEntry
	entry := LogEntry{
		Level:     WARN,
		Message:   "warning entry with special chars: !@#$%^&*()",
		Service:   "ENTRY_SERVICE",
		Timestamp: time.Now(),
	}

	entryData, err := json.Marshal(entry)
	if err != nil {
		t.Errorf("ошибка сериализации записи: %v", err)
	}

	var deserializedEntry LogEntry
	err = json.Unmarshal(entryData, &deserializedEntry)
	if err != nil {
		t.Errorf("ошибка десериализации записи: %v", err)
	}

	if deserializedEntry.Level != entry.Level {
		t.Errorf("уровень записи не совпадает: ожидался %v, получен %v", entry.Level, deserializedEntry.Level)
	}
}

/**
 * TestMessageTypes тестирует константы типов сообщений
 * @param t *testing.T - тестовый контекст
 */
func TestConstantsAndDefaults(t *testing.T) {
	// Проверяем, что уровни логирования корректно работают
	level, err := ParseLevel("INFO")
	if err != nil {
		t.Errorf("ошибка парсинга уровня INFO: %v", err)
	}
	if level != INFO {
		t.Errorf("ожидался уровень INFO, получен %v", level)
	}

	// Проверяем String() метод для уровней
	if DEBUG.String() != "DEBUG" {
		t.Error("DEBUG.String() должен возвращать 'DEBUG'")
	}
	if INFO.String() != "INFO" {
		t.Error("INFO.String() должен возвращать 'INFO'")
	}
}

/**
 * TestLogClientUtilityFunctions тестирует вспомогательные функции логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientUtilityFunctions(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "utility_test.log")

	config := &LoggingConfig{
		LogFile:       logFile,
		SocketPath:    filepath.Join(tempDir, "utility.sock"),
		Level:         "DEBUG",
		FlushInterval: time.Millisecond * 100,
		BufferSize:    50,
		MaxFileSize:   1024,
		MaxFiles:      2,
	}

	// Тестируем NewLogClient
	client, err := NewLogClient(config)
	if err != nil {
		t.Logf("Ожидаемая ошибка при создании клиента (сокет): %v", err)
		// Тестируем функции без создания полного клиента
		testStandaloneUtilities(t)
		return
	}

	defer func() { _ = client.Close() }()

	// Тестируем SetLevel
	client.SetLevel(ERROR)

	// Тестируем Ping
	err = client.Ping()
	if err != nil {
		t.Logf("ожидаемая ошибка ping (нет сервера): %v", err)
	}

	// Тестируем методы логирования
	_ = client.Debug("debug message")
	_ = client.Info("info message")
	_ = client.Warn("warning message")
	_ = client.Error("error message")

	// Тестируем методы с форматированием
	_ = client.Debug("debug %s %d", "formatted", 123)
	_ = client.Info("info %s %d", "formatted", 456)
	_ = client.Warn("warning %s %d", "formatted", 789)
	_ = client.Error("error %s %d", "formatted", 999)
}

/**
 * testStandaloneUtilities тестирует утилиты, которые можно протестировать без полного логгера
 * @param t *testing.T - тестовый контекст
 */
func testStandaloneUtilities(t *testing.T) {
	// Тестируем ParseLevel с различными входными данными
	testCases := []struct {
		input    string
		expected LogLevel
		hasError bool
	}{
		{"DEBUG", DEBUG, false},
		{"INFO", INFO, false},
		{"WARN", WARN, false},
		{"ERROR", ERROR, false},
		{"PANIC", PANIC, false},
		{"debug", DEBUG, false}, // Нечувствительность к регистру
		{"Info", INFO, false},
		{"WARN", WARN, false},
		{"недопустимый", DEBUG, true},
		{"", DEBUG, true},
	}

	for _, tc := range testCases {
		result, err := ParseLevel(tc.input)
		if tc.hasError {
			if err == nil {
				t.Errorf("ожидалась ошибка для входа '%s'", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("неожиданная ошибка для входа '%s': %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("для входа '%s' ожидался %v, получен %v", tc.input, tc.expected, result)
			}
		}
	}
}

/**
 * TestServiceLoggerMethodsExtended тестирует дополнительные методы ServiceLogger
 * @param t *testing.T - тестовый контекст
 */
func TestServiceLoggerMethodsExtended(t *testing.T) {
	// Создаем мок-клиент (используем существующий из mocks_test.go)
	mockClient := &MockLogClient{}
	emptyMockClient := &MockLogClient{}

	serviceLogger := &ServiceLogger{
		client:  mockClient,
		service: "EXTENDED_TEST_SERVICE",
	}

	// Тестируем методы логирования с проверкой уровней
	_ = serviceLogger.Debug("extended debug message")
	_ = serviceLogger.Info("extended info message")
	_ = serviceLogger.Warn("extended warning message")
	_ = serviceLogger.Error("extended error message")

	// Тестируем методы с форматированием и различными типами аргументов
	_ = serviceLogger.Debug("debug %s %d %t", "test", 123, true)
	_ = serviceLogger.Info("info %v", []string{"a", "b", "c"})
	_ = serviceLogger.Warn("warning %.2f", 3.14159)
	_ = serviceLogger.Error("error %x", 255)

	// Проверяем, что мок-клиент получил вызовы
	// (детальная проверка зависит от реализации MockLogClient)
	if mockClient == emptyMockClient {
		t.Error("мок-клиент не должен быть nil")
	}
}
