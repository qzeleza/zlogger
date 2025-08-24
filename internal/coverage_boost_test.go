package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

/**
 * TestLogServerConfigValidationBoost тестирует валидацию конфигурации сервера без сокетов
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerConfigValidationBoost(t *testing.T) {
	testCases := []struct {
		name        string
		config      *LoggingConfig
		expectError bool
	}{
		{
			name:        "nil конфигурация",
			config:      nil,
			expectError: true,
		},
		{
			name: "пустой путь к файлу",
			config: &LoggingConfig{
				LogFile:    "",
				SocketPath: "/tmp/test.sock",
				Level:      "INFO",
			},
			expectError: true,
		},
		{
			name: "пустой путь к сокету",
			config: &LoggingConfig{
				LogFile:    "/tmp/test.log",
				SocketPath: "",
				Level:      "INFO",
			},
			expectError: true,
		},
		{
			name: "невалидный уровень логирования",
			config: &LoggingConfig{
				LogFile:    "/tmp/test.log",
				SocketPath: "/tmp/test.sock",
				Level:      "НЕДОПУСТИМЫЙ_УРОВЕНЬ",
			},
			expectError: true,
		},
	}

	// Тестируем только валидацию конфигурации, не создавая реальные сокеты
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewLogServer(tc.config)
			if tc.expectError {
				if err == nil {
					t.Errorf("ожидалась ошибка для случая %s", tc.name)
				}
			} else {
				// Для валидных конфигураций ожидаем ошибку сокета, но не валидации
				if err != nil && !strings.Contains(err.Error(), "сокет") {
					t.Errorf("неожиданная ошибка валидации для случая %s: %v", tc.name, err)
				}
			}
		})
	}
}

/**
 * TestLogServerHelperMethodsBoost тестирует вспомогательные методы сервера без сокетов
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerHelperMethodsBoost(t *testing.T) {
	// Тестируем функции форматирования и парсинга без создания полного сервера
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "helper_test.log")

	// Создаем файл для тестирования
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}
	defer file.Close()

	// Создаем минимальную структуру сервера для тестирования методов
	server := &LogServer{
		file:        file,
		currentSize: 0,
		stats:       ServerStats{StartTime: time.Now()},
		config: &LoggingConfig{
			MaxFileSize: 1024 * 1024, // 1MB
			MaxFiles:    3,
		},
	}

	// Тестируем formatMessageAsTXT
	msg := LogMessage{
		Level:     WARN,
		Message:   "test warning message",
		Service:   "HELPER_TEST",
		Timestamp: time.Now(),
		ClientID:  "test-client",
	}

	formatted := server.formatMessageAsTXT(msg)
	if !strings.Contains(formatted, "test warning message") {
		t.Error("отформатированное сообщение должно содержать исходный текст")
	}
	if !strings.Contains(formatted, "HELPER_TEST") {
		t.Error("отформатированное сообщение должно содержать имя сервиса")
	}
	if !strings.Contains(formatted, "WARN") {
		t.Error("отформатированное сообщение должно содержать уровень логирования")
	}

	// Тестируем writeMessage
	server.writeMessage(msg)
	if server.stats.TotalMessages == 0 {
		t.Error("счетчик сообщений должен увеличиться")
	}

	// Тестируем rotateIfNeeded с MaxFiles = 1 (простая очистка)
	server.config = &LoggingConfig{
		LogFile:     logFile,
		MaxFiles:    1,
		MaxFileSize: 1024,
	}
	server.currentSize = 2000 // Превышаем лимит

	err = server.rotateIfNeeded()
	if err != nil {
		t.Errorf("ошибка ротации: %v", err)
	}
	if server.currentSize != 0 {
		t.Error("размер файла должен быть сброшен после ротации")
	}
}

/**
 * TestLogServerStatisticsBoost тестирует статистику сервера
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerStatisticsBoost(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "stats_test.log")

	// Создаем файл для тестирования
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}
	defer file.Close()

	// Создаем сервер с базовой статистикой
	server := &LogServer{
		file:        file,
		currentSize: 0,
		stats: ServerStats{
			StartTime:     time.Now(),
			TotalMessages: 0,
			TotalClients:  0,
			FileRotations: 0,
		},
		config: &LoggingConfig{
			MaxFileSize: 1024 * 1024, // 1MB
			MaxFiles:    3,
		},
	}

	// Тестируем увеличение счетчиков
	msg := LogMessage{
		Level:     INFO,
		Message:   "test stats message",
		Service:   "STATS_TEST",
		Timestamp: time.Now(),
		ClientID:  "stats-client",
	}

	// Записываем несколько сообщений
	for i := 0; i < 5; i++ {
		server.writeMessage(msg)
	}

	if server.stats.TotalMessages != 5 {
		t.Errorf("ожидалось 5 сообщений, получено %d", server.stats.TotalMessages)
	}

	// Тестируем parseLogEntry вместо handlePing (избегаем nil encoder)
	logLine1 := "[STATS] 26-01-2025 12:00:00 [INFO ] \"test stats message\""
	_, parseErr := server.parseLogEntry(logLine1)
	if parseErr != nil {
		t.Errorf("ошибка парсинга валидной строки: %v", parseErr)
	}

	// Тестируем parseLogEntry
	logLine2 := "[TEST ] 26-01-2025 12:00:00 [INFO ] \"parsed message\""
	entry, err := server.parseLogEntry(logLine2)
	if err != nil {
		t.Errorf("ошибка парсинга лог-записи: %v", err)
	}
	if entry.Service != "TEST" {
		t.Errorf("ожидался сервис TEST, получен %s", entry.Service)
	}
	if entry.Level != INFO {
		t.Errorf("ожидался уровень INFO, получен %s", entry.Level)
	}
	if entry.Message != "parsed message" {
		t.Errorf("ожидалось сообщение 'parsed message', получено %s", entry.Message)
	}
}

/**
 * TestLogServerMessageProcessingBoost тестирует обработку различных типов сообщений
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerMessageProcessingBoost(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "processing_test.log")

	// Создаем файл для тестирования
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}
	defer file.Close()

	server := &LogServer{
		file:        file,
		currentSize: 0,
		stats:       ServerStats{StartTime: time.Now()},
		config: &LoggingConfig{
			MaxFileSize: 1024 * 1024, // 1MB
			MaxFiles:    3,
		},
	}

	// Тестируем parseLogEntry с различными форматами
	validLogLines := []string{
		"[TEST ] 26-01-2025 12:00:00 [INFO ] \"valid info message\"",
		"[SERV ] 26-01-2025 13:30:45 [ERROR] \"error occurred\"",
		"[DEBUG] 26-01-2025 14:15:30 [DEBUG] \"debug information\"",
	}

	for _, line := range validLogLines {
		entry, err := server.parseLogEntry(line)
		if err != nil {
			t.Errorf("ошибка парсинга валидной строки '%s': %v", line, err)
			continue
		}
		if entry.Service == "" {
			t.Errorf("сервис не должен быть пустым для строки: %s", line)
		}
		if entry.Message == "" {
			t.Errorf("сообщение не должно быть пустым для строки: %s", line)
		}
	}

	// Тестируем formatMessageAsTXT с различными уровнями
	levels := []LogLevel{DEBUG, INFO, WARN, ERROR, PANIC}
	for _, level := range levels {
		msg := LogMessage{
			Level:     level,
			Message:   fmt.Sprintf("test %s message", level),
			Service:   "LEVEL_TEST",
			Timestamp: time.Now(),
			ClientID:  "level-client",
		}

		formatted := server.formatMessageAsTXT(msg)
		if !strings.Contains(formatted, level.String()) {
			t.Errorf("отформатированное сообщение должно содержать уровень %s", level.String())
		}
		if !strings.Contains(formatted, "LEVEL_TEST") {
			t.Errorf("отформатированное сообщение должно содержать имя сервиса")
		}
	}
}

/**
 * TestLogServerBufferHandling тестирует обработку буферов
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerBufferHandling(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "buffer_test.log")

	// Создаем файл для тестирования
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}
	defer file.Close()

	// Создаем сервер с базовой конфигурацией
	server := &LogServer{
		file:        file,
		currentSize: 0,
		stats:       ServerStats{StartTime: time.Now()},
		config: &LoggingConfig{
			BufferSize:    3,
			FlushInterval: time.Millisecond * 100,
		},
	}

	// Тестируем добавление сообщений в буфер
	msg1 := LogMessage{
		Level:     INFO,
		Message:   "buffer message 1",
		Service:   "BUFFER_TEST",
		Timestamp: time.Now(),
		ClientID:  "buffer-client",
	}

	msg2 := LogMessage{
		Level:     WARN,
		Message:   "buffer message 2",
		Service:   "BUFFER_TEST",
		Timestamp: time.Now(),
		ClientID:  "buffer-client",
	}

	// Добавляем сообщения
	server.writeMessage(msg1)
	server.writeMessage(msg2)

	// Проверяем, что сообщения были обработаны (минимум 1)
	if server.stats.TotalMessages < 1 {
		t.Errorf("ожидалось минимум 1 сообщение в статистике, получено %d", server.stats.TotalMessages)
	}

	// Тестируем обработку различных сообщений
	pingMsg := LogMessage{
		Level:     INFO,
		Message:   "PING",
		Service:   "BUFFER_TEST",
		Timestamp: time.Now(),
		ClientID:  "ping-client",
	}

	server.writeMessage(pingMsg)

	// Тестируем debug сообщение
	levelMsg := LogMessage{
		Level:     DEBUG,
		Message:   "debug message",
		Service:   "BUFFER_TEST",
		Timestamp: time.Now(),
		ClientID:  "level-client",
	}

	server.writeMessage(levelMsg)
}

/**
 * TestLogServerEdgeCases тестирует граничные случаи
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "edge_test.log")

	// Создаем файл для тестирования
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}
	defer file.Close()

	server := &LogServer{
		file:        file,
		currentSize: 0,
		stats:       ServerStats{StartTime: time.Now()},
		config: &LoggingConfig{
			MaxFileSize: 1024 * 1024, // 1MB
			MaxFiles:    3,
		},
	}

	// Тестируем пустое сообщение
	emptyMsg := LogMessage{
		Level:     INFO,
		Message:   "",
		Service:   "EDGE_TEST",
		Timestamp: time.Now(),
		ClientID:  "edge-client",
	}

	server.writeMessage(emptyMsg)

	// Тестируем очень длинное сообщение
	longMessage := strings.Repeat("A", 1000)
	longMsg := LogMessage{
		Level:     ERROR,
		Message:   longMessage,
		Service:   "EDGE_TEST",
		Timestamp: time.Now(),
		ClientID:  "edge-client",
	}

	server.writeMessage(longMsg)

	// Тестируем сообщение с специальными символами
	specialMsg := LogMessage{
		Level:     WARN,
		Message:   "сообщение с юникодом: 🚀 и символами \"quotes\" и \n новые строки",
		Service:   "EDGE_TEST",
		Timestamp: time.Now(),
		ClientID:  "edge-client",
	}

	server.writeMessage(specialMsg)

	if server.stats.TotalMessages != 3 {
		t.Errorf("ожидалось 3 сообщения, получено %d", server.stats.TotalMessages)
	}

	// Тестируем parseLogEntry с невалидными данными
	invalidLines := []string{
		"", // пустая строка
		"недопустимая строка лога без правильного формата",
		"[TEST] недопустимый формат времени",
		"[TEST ] 2025-01-26 12:00:00 [НЕДОПУСТИМЫЙ_УРОВЕНЬ] \"message\"",
	}

	for _, line := range invalidLines {
		_, err := server.parseLogEntry(line)
		if err == nil {
			t.Errorf("ожидалась ошибка для невалидной строки: %s", line)
		}
	}
}
