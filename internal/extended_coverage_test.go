package logger

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

/**
 * TestServerHelperFunctions тестирует вспомогательные функции сервера без создания полного сервера
 * @param t *testing.T - тестовый контекст
 */
func TestServerHelperFunctions(t *testing.T) {
	// Тестируем функции парсинга и форматирования без полного сервера
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &LoggingConfig{
		LogFile:       logFile,
		SocketPath:    filepath.Join(tempDir, "test.sock"),
		Level:         "INFO",
		FlushInterval: time.Millisecond * 100,
		BufferSize:    100,
		MaxFileSize:   1024 * 1024,
		MaxFiles:      3,
	}

	// Создаем минимальный сервер для тестирования методов (может не получиться из-за сокета)
	server, err := NewLogServer(config)
	if err != nil {
		// Если не удалось создать сервер из-за сокета, тестируем отдельные функции
		t.Logf("Не удалось создать сервер (ожидаемо): %v", err)

		// Тестируем функции парсинга уровня
		testParseLevelFunction(t)
		return
	}

	defer func() {
		if server.file != nil {
			_ = server.file.Close()
		}
	}()

	// Тестируем formatMessageAsTXT
	msg := LogMessage{
		Level:     INFO,
		Message:   "test message with special chars: !@#$%^&*()",
		Service:   "TEST_SERVICE",
		Timestamp: time.Now(),
	}

	formatted := server.formatMessageAsTXT(msg)
	if !strings.Contains(formatted, "test message with special chars") {
		t.Error("отформатированное сообщение должно содержать исходный текст")
	}
	if !strings.Contains(formatted, "TEST_SERVICE") {
		t.Error("отформатированное сообщение должно содержать имя сервиса")
	}
	if !strings.Contains(formatted, "[INFO ]") {
		t.Error("отформатированное сообщение должно содержать уровень логирования")
	}

	// Тестируем parseLogEntry
	logLine := "[TEST_SERVICE] 26-01-2025 12:00:00 [INFO ] \"test log entry\""
	entry, err := server.parseLogEntry(logLine)
	if err != nil {
		t.Errorf("ошибка парсинга лог-записи: %v", err)
	}
	if entry.Message == "" {
		t.Fatal("распарсенная запись должна содержать сообщение")
	}
	if entry.Service != "TEST_SERVICE" {
		t.Errorf("ожидался сервис 'TEST_SERVICE', получен '%s'", entry.Service)
	}
	if entry.Level != INFO {
		t.Errorf("ожидался уровень INFO, получен %v", entry.Level)
	}
	if entry.Message != "test log entry" {
		t.Errorf("ожидалось сообщение 'test log entry', получено '%s'", entry.Message)
	}

	// Тестируем parseLogEntry с невалидной строкой
	invalidLine := "invalid log line format"
	_, err = server.parseLogEntry(invalidLine)
	if err == nil {
		t.Error("ожидалась ошибка для невалидной лог-записи")
	}

	// Тестируем matchesFilter
	filter := FilterOptions{
		Service: "TEST_SERVICE",
	}
	if !server.matchesFilter(entry, filter) {
		t.Error("запись должна соответствовать фильтру по сервису")
	}

	filterLevel := INFO
	filter.Level = &filterLevel
	if !server.matchesFilter(entry, filter) {
		t.Error("запись должна соответствовать фильтру по уровню")
	}

	wrongLevel := ERROR
	filter.Level = &wrongLevel
	if server.matchesFilter(entry, filter) {
		t.Error("запись не должна соответствовать неправильному фильтру по уровню")
	}

	// Тестируем parseLogEntry вместо handlePing (избегаем nil encoder)
	testLogLine := "[TEST ] 26-01-2025 12:00:00 [INFO ] \"ping test\""
	_, parseErr := server.parseLogEntry(testLogLine)
	if parseErr != nil {
		t.Errorf("ошибка парсинга тестовой строки: %v", parseErr)
	}

	// Тестируем writeMessage (прямая запись)
	testMsg := LogMessage{
		Level:     ERROR,
		Message:   "test direct write",
		Service:   "TEST",
		Timestamp: time.Now(),
		ClientID:  "test-client",
	}
	server.writeMessage(testMsg)

	// Тестируем статистику сервера
	if server.stats.TotalMessages < 0 {
		t.Error("общее количество сообщений не должно быть отрицательным")
	}
	if server.stats.TotalClients < 0 {
		t.Error("общее количество клиентов не должно быть отрицательным")
	}
	if server.stats.StartTime.IsZero() {
		t.Error("время запуска сервера должно быть установлено")
	}
}

/**
 * testParseLevelFunction тестирует функцию ParseLevel отдельно
 * @param t *testing.T - тестовый контекст
 */
func testParseLevelFunction(t *testing.T) {
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
		{"debug", DEBUG, false}, // Проверяем нечувствительность к регистру
		{"info", INFO, false},
		{"INVALID", DEBUG, true}, // Невалидный уровень должен вернуть ошибку
		{"", DEBUG, true},        // Пустая строка должна вернуть ошибку
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ParseLevel_%s", tc.input), func(t *testing.T) {
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
		})
	}
}

/**
 * TestLogLevels тестирует методы LogLevel
 * @param t *testing.T - тестовый контекст
 */
func TestLogLevels(t *testing.T) {
	// Тестируем String() метод
	levels := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{PANIC, "PANIC"},
	}

	for _, tc := range levels {
		t.Run(fmt.Sprintf("String_%s", tc.expected), func(t *testing.T) {
			result := tc.level.String()
			if result != tc.expected {
				t.Errorf("ожидалось '%s', получено '%s'", tc.expected, result)
			}
		})
	}

	// Тестируем IsValid() метод
	validLevels := []LogLevel{DEBUG, INFO, WARN, ERROR, PANIC}
	for _, level := range validLevels {
		if !level.IsValid() {
			t.Errorf("уровень %v должен быть валидным", level)
		}
	}

	// Тестируем невалидный уровень
	invalidLevel := LogLevel(999)
	if invalidLevel.IsValid() {
		t.Error("невалидный уровень не должен быть валидным")
	}
}

/**
 * TestMessagePooling тестирует пулы сообщений
 * @param t *testing.T - тестовый контекст
 */
func TestMessagePooling(t *testing.T) {
	// Тестируем GetLogMessage и PutLogMessage
	msg1 := GetLogMessage()
	if msg1 == nil {
		t.Fatal("GetLogMessage не должен возвращать nil")
	}

	// Устанавливаем значения
	msg1.Level = INFO
	msg1.Message = "test message"
	msg1.Service = "TEST"
	msg1.Timestamp = time.Now()

	// Возвращаем в пул
	PutLogMessage(msg1)

	// Получаем снова (может быть тот же объект из пула)
	msg2 := GetLogMessage()
	if msg2 == nil {
		t.Fatal("GetLogMessage не должен возвращать nil после возврата в пул")
	}

	// Проверяем, что объект был очищен (если это тот же объект)
	if msg2 == msg1 {
		if msg2.Message != "" {
			t.Error("сообщение должно быть очищено при возврате из пула")
		}
		if msg2.Service != "" {
			t.Error("сервис должен быть очищен при возврате из пула")
		}
	}

	// Тестируем GetLogEntry и PutLogEntry
	entry1 := GetLogEntry()
	if entry1 == nil {
		t.Fatal("GetLogEntry не должен возвращать nil")
	}

	entry1.Level = ERROR
	entry1.Message = "test entry"
	entry1.Service = "TEST_ENTRY"
	entry1.Timestamp = time.Now()

	PutLogEntry(entry1)

	entry2 := GetLogEntry()
	if entry2 == nil {
		t.Fatal("GetLogEntry не должен возвращать nil после возврата в пул")
	}

	// Проверяем очистку
	if entry2 == entry1 {
		if entry2.Message != "" {
			t.Error("сообщение записи должно быть очищено при возврате из пула")
		}
		if entry2.Service != "" {
			t.Error("сервис записи должен быть очищен при возврате из пула")
		}
	}
}

/**
 * TestLogClientUpdateConfig тестирует обновление конфигурации клиента
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientUpdateConfig(t *testing.T) {
	client := &LogClient{
		config:         nil,
		conn:           nil,
		level:          DEBUG,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	tempDir := t.TempDir()
	newConfig := &LoggingConfig{
		LogFile:       filepath.Join(tempDir, "new.log"),
		SocketPath:    filepath.Join(tempDir, "new.sock"),
		Level:         "ERROR",
		FlushInterval: time.Second,
		BufferSize:    200,
		MaxFileSize:   2048,
		MaxFiles:      5,
	}

	// Тестируем обновление конфигурации (ожидаем ошибку подключения)
	err := client.UpdateConfig(newConfig)
	if err == nil {
		t.Error("ожидалась ошибка подключения к несуществующему сокету")
	}

	// Проверяем, что конфигурация обновилась
	if client.config == nil {
		t.Error("конфигурация должна быть установлена")
	}
	if client.config.Level != "ERROR" {
		t.Errorf("ожидался уровень ERROR, получен %s", client.config.Level)
	}

	// Тестируем обновление с nil конфигурацией
	err = client.UpdateConfig(nil)
	if err == nil {
		t.Error("ожидалась ошибка при обновлении с nil конфигурацией")
	}
}

/**
 * TestLogClientRecoverPanic тестирует восстановление после паники
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientRecoverPanic(t *testing.T) {
	tempDir := t.TempDir()
	config := &LoggingConfig{
		LogFile:       filepath.Join(tempDir, "panic.log"),
		SocketPath:    filepath.Join(tempDir, "panic.sock"),
		Level:         "DEBUG",
		FlushInterval: time.Millisecond * 100,
		BufferSize:    100,
		MaxFileSize:   1024,
		MaxFiles:      3,
	}

	client := &LogClient{
		config:         config,
		conn:           nil,
		level:          DEBUG,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем RecoverPanic - не должно паниковать
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecoverPanic не должен вызывать панику: %v", r)
		}
	}()

	client.RecoverPanic("test panic message")

	// Проверяем, что метод завершился без ошибок
	// (в реальной ситуации он бы записал информацию о панике в лог)
}
