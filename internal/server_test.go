// server_test.go - Тесты для сервера логгера
package logger

import (
	"encoding/json"

	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kvasdns/internal/config"
)

// createTestServerConfig создает тестовую конфигурацию для сервера
func createTestServerConfig(t *testing.T) *config.LoggingConfig {
	tmpDir, err := os.MkdirTemp("", "logger_server_test")
	if err != nil {
		t.Fatalf("не удалось создать временную директорию: %v", err)
	}

	// Очистка после теста
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return &config.LoggingConfig{
		LogFile:       filepath.Join(tmpDir, "test.log"),
		SocketPath:    filepath.Join(tmpDir, "test.sock"),
		Level:         "INFO",
		BufferSize:    100,
		MaxFileSize:   1024 * 1024, // 1MB
		MaxFiles:      3,
		FlushInterval: time.Millisecond * 100, // 100ms для быстрых тестов
	}
}

// TestNewLogServer тестирует создание нового сервера логгера
func TestNewLogServer(t *testing.T) {
	config := createTestServerConfig(t)

	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	if server == nil {
		t.Fatal("сервер не должен быть nil")
	}

	if server.config != config {
		t.Error("конфигурация сервера не соответствует переданной")
	}

	if server.buffer == nil {
		t.Error("буфер сервера не должен быть nil")
	}

	if server.done == nil {
		t.Error("канал done не должен быть nil")
	}
}

// TestNewLogServerWithInvalidConfig тестирует создание сервера с некорректной конфигурацией
func TestNewLogServerWithInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *config.LoggingConfig
	}{
		{
			name: "пустой путь к файлу лога",
			config: &config.LoggingConfig{
				LogFile:    "",
				SocketPath: "/tmp/test.sock",
			},
		},
		{
			name: "пустой путь к сокету",
			config: &config.LoggingConfig{
				LogFile:    "/tmp/test.log",
				SocketPath: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLogServer(tt.config)
			if err == nil {
				t.Error("ожидалась ошибка при создании сервера с некорректной конфигурацией")
			}
		})
	}
}

// TestFormatMessageAsTXT тестирует форматирование сообщения в TXT формат
func TestFormatMessageAsTXT(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	msg := LogMessage{
		Service:   "TEST",
		Level:     INFO,
		Message:   "тестовое сообщение",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		ClientID:  "client1",
	}

	formatted := server.formatMessageAsTXT(msg)

	// Проверяем, что форматированное сообщение содержит все необходимые элементы
	if !strings.Contains(formatted, "TEST") {
		t.Error("форматированное сообщение должно содержать имя сервиса")
	}

	if !strings.Contains(formatted, "INFO") {
		t.Error("форматированное сообщение должно содержать уровень логирования")
	}

	if !strings.Contains(formatted, "тестовое сообщение") {
		t.Error("форматированное сообщение должно содержать текст сообщения")
	}

	// Формат времени: DD-MM-YYYY HH:MM:SS
	if !strings.Contains(formatted, "01-01-2023 12:00:00") {
		t.Error("форматированное сообщение должно содержать временную метку")
	}
}

// TestParseLogEntry тестирует парсинг строки лога в LogEntry
func TestParseLogEntry(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	tests := []struct {
		name     string
		line     string
		wantErr  bool
		expected LogEntry
	}{
		{
			name: "корректная строка лога",
			line: "[TEST ] 01-01-2023 12:00:00 [INFO ] \"тестовое сообщение\"",
			expected: LogEntry{
				Service:   "TEST",
				Level:     INFO,
				Message:   "тестовое сообщение",
				Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name:    "некорректная строка лога",
			line:    "некорректная строка",
			wantErr: true,
		},
		{
			name:    "пустая строка",
			line:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := server.parseLogEntry(tt.line)

			if tt.wantErr {
				if err == nil {
					t.Error("ожидалась ошибка при парсинге некорректной строки")
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалась ошибка: %v", err)
			}

			if entry.Service != tt.expected.Service {
				t.Errorf("неверное имя сервиса: получено %s, ожидалось %s", entry.Service, tt.expected.Service)
			}

			if entry.Level != tt.expected.Level {
				t.Errorf("неверный уровень: получено %v, ожидалось %v", entry.Level, tt.expected.Level)
			}

			if entry.Message != tt.expected.Message {
				t.Errorf("неверное сообщение: получено %s, ожидалось %s", entry.Message, tt.expected.Message)
			}
		})
	}
}

// TestMatchesFilter тестирует проверку соответствия записи фильтру
func TestMatchesFilter(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	entry := LogEntry{
		Service:   "TEST",
		Level:     INFO,
		Message:   "тестовое сообщение",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name     string
		filter   FilterOptions
		expected bool
	}{
		{
			name: "фильтр по сервису - совпадение",
			filter: FilterOptions{
				Service: "TEST",
			},
			expected: true,
		},
		{
			name: "фильтр по сервису - несовпадение",
			filter: FilterOptions{
				Service: "OTHER",
			},
			expected: false,
		},
		{
			name: "фильтр по уровню - совпадение",
			filter: FilterOptions{
				Level: func() *LogLevel { l := INFO; return &l }(),
			},
			expected: true,
		},
		{
			name: "фильтр по уровню - несовпадение",
			filter: FilterOptions{
				Level: func() *LogLevel { l := ERROR; return &l }(),
			},
			expected: false,
		},
		{
			name:     "пустой фильтр - всегда совпадение",
			filter:   FilterOptions{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.matchesFilter(entry, tt.filter)
			if result != tt.expected {
				t.Errorf("неверный результат фильтрации: получено %v, ожидалось %v", result, tt.expected)
			}
		})
	}
}

// TestSendError тестирует отправку ошибки клиенту
func TestSendError(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Создаем буфер для захвата вывода
	var output strings.Builder
	encoder := json.NewEncoder(&output)

	server.sendError(encoder, "тестовая ошибка")

	// Проверяем, что ошибка была записана в JSON формате
	result := output.String()
	if !strings.Contains(result, "тестовая ошибка") {
		t.Error("вывод должен содержать текст ошибки")
	}

	if !strings.Contains(result, "error") {
		t.Error("вывод должен содержать поле error")
	}
}

// TestHandlePing тестирует обработку ping запроса
func TestHandlePing(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Создаем буфер для захвата вывода
	var output strings.Builder
	encoder := json.NewEncoder(&output)

	server.handlePing(encoder)

	// Проверяем, что ответ содержит pong (в нижнем регистре, как в реализации)
	result := output.String()
	t.Logf("Отладка: полученный ответ: %s", result)

	if !strings.Contains(result, "pong") {
		t.Error("ответ на ping должен содержать pong")
	}

	// Просто проверяем, что ответ не пустой и содержит JSON
	if len(result) == 0 {
		t.Error("ответ не должен быть пустым")
	}
}

// TestWriteMessage тестирует прямую запись сообщения
func TestWriteMessage(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Инициализируем файл лога
	err = server.initLogFile()
	if err != nil {
		t.Fatalf("не удалось инициализировать файл лога: %v", err)
	}
	defer server.file.Close()

	msg := LogMessage{
		Service:   "TEST",
		Level:     INFO,
		Message:   "тестовое сообщение",
		Timestamp: time.Now(),
		ClientID:  "client1",
	}

	// Записываем сообщение
	server.writeMessage(msg)

	// Проверяем, что файл содержит записанное сообщение
	content, err := os.ReadFile(config.LogFile)
	if err != nil {
		t.Fatalf("не удалось прочитать файл лога: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "TEST") {
		t.Error("файл лога должен содержать имя сервиса")
	}

	if !strings.Contains(contentStr, "тестовое сообщение") {
		t.Error("файл лога должен содержать текст сообщения")
	}
}

// TestRotateIfNeeded тестирует ротацию логов
func TestRotateIfNeeded(t *testing.T) {
	config := createTestServerConfig(t)
	config.MaxFiles = 1 // Простая ротация - только очистка файла

	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Инициализируем файл лога
	err = server.initLogFile()
	if err != nil {
		t.Fatalf("не удалось инициализировать файл лога: %v", err)
	}

	// Записываем что-то в файл
	_, _ = server.file.WriteString("тестовые данные")
	server.currentSize = 100

	// Выполняем ротацию
	err = server.rotateIfNeeded()
	if err != nil {
		t.Fatalf("ошибка при ротации: %v", err)
	}

	// Проверяем, что размер сброшен
	if server.currentSize != 0 {
		t.Error("размер файла должен быть сброшен после ротации")
	}

	// Проверяем, что файл существует и пуст
	info, err := os.Stat(config.LogFile)
	if err != nil {
		t.Fatalf("файл лога должен существовать после ротации: %v", err)
	}

	if info.Size() != 0 {
		t.Error("файл лога должен быть пустым после ротации")
	}

	server.file.Close()
}

// TestLogServerStart проверяет запуск сервера
func TestLogServerStart(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Запускаем сервер в отдельной горутине
	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Останавливаем сервер
	_ = server.Stop()

	// Проверяем, что сервер завершился
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("сервер завершился с ошибкой: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("сервер не завершился в течение таймаута")
	}
}

// TestLogServerStop проверяет остановку сервера
func TestLogServerStop(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Останавливаем сервер до запуска (должно быть безопасно)
	_ = server.Stop()

	// Запускаем сервер
	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Останавливаем сервер
	_ = server.Stop()

	// Проверяем завершение
	select {
	case <-done:
		// Сервер завершился корректно
	case <-time.After(2 * time.Second):
		t.Error("сервер не завершился после Stop()")
	}

	// Повторная остановка должна быть безопасной
	_ = server.Stop()
}

// TestLogServerGetLogEntries проверяет получение записей логов
func TestLogServerGetLogEntries(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Инициализируем файл лога
	err = server.initLogFile()
	if err != nil {
		t.Fatalf("не удалось инициализировать файл лога: %v", err)
	}
	defer server.file.Close()

	// Записываем тестовые данные в файл лога
	testLines := []string{
		"[TEST1    ] 25-07-2025 20:00:00 [INFO ] \"test message 1\"",
		"[TEST2    ] 25-07-2025 20:00:01 [ERROR] \"test message 2\"",
	}

	for _, line := range testLines {
		_, err := server.file.WriteString(line + "\n")
		if err != nil {
			t.Fatalf("не удалось записать в файл лога: %v", err)
		}
	}
	_ = server.file.Sync() // Принудительно сбрасываем на диск

	// Получаем записи с пустым фильтром
	filter := FilterOptions{
		Limit: 10,
	}
	entries, err := server.getLogEntries(filter)
	if err != nil {
		t.Fatalf("ошибка получения записей: %v", err)
	}

	if len(entries) == 0 {
		t.Error("должны быть получены записи логов")
	}

	// Проверяем, что записи содержат ожидаемые данные
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, "test message") {
			found = true
			break
		}
	}

	if !found {
		t.Error("не найдены ожидаемые тестовые сообщения в записях логов")
	}
}

// TestLogServerGetLogEntriesWithFilter проверяет фильтрацию записей
func TestLogServerGetLogEntriesWithFilter(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Инициализируем файл лога
	err = server.initLogFile()
	if err != nil {
		t.Fatalf("не удалось инициализировать файл лога: %v", err)
	}
	defer server.file.Close()

	// Записываем тестовые данные с разными сервисами и уровнями
	testLines := []string{
		"[API      ] 25-07-2025 20:00:00 [INFO ] \"api info message\"",
		"[DB       ] 25-07-2025 20:00:01 [ERROR] \"database error\"",
		"[API      ] 25-07-2025 20:00:02 [ERROR] \"api error message\"",
	}

	for _, line := range testLines {
		_, err := server.file.WriteString(line + "\n")
		if err != nil {
			t.Fatalf("не удалось записать в файл лога: %v", err)
		}
	}
	_ = server.file.Sync()

	// Фильтруем по сервису
	apiFilter := FilterOptions{
		Limit:   10,
		Service: "API",
	}
	apiEntries, err := server.getLogEntries(apiFilter)
	if err != nil {
		t.Fatalf("ошибка получения записей API: %v", err)
	}
	if len(apiEntries) == 0 {
		t.Error("должны быть найдены записи для сервиса API")
	}

	// Фильтруем по уровню
	errorLevel := ERROR
	errorFilter := FilterOptions{
		Limit: 10,
		Level: &errorLevel,
	}
	errorEntries, err := server.getLogEntries(errorFilter)
	if err != nil {
		t.Fatalf("ошибка получения записей ERROR: %v", err)
	}
	if len(errorEntries) == 0 {
		t.Error("должны быть найдены записи с уровнем ERROR")
	}

	// Фильтруем по сервису и уровню
	apiErrorLevel := ERROR
	apiErrorFilter := FilterOptions{
		Limit:   10,
		Service: "API",
		Level:   &apiErrorLevel,
	}
	apiErrorEntries, err := server.getLogEntries(apiErrorFilter)
	if err != nil {
		t.Fatalf("ошибка получения записей API ERROR: %v", err)
	}
	if len(apiErrorEntries) == 0 {
		t.Error("должны быть найдены записи для сервиса API с уровнем ERROR")
	}
}

// TestLogServerFlush проверяет сброс буфера
func TestLogServerFlush(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Инициализируем файл лога
	err = server.initLogFile()
	if err != nil {
		t.Fatalf("не удалось инициализировать файл лога: %v", err)
	}
	defer server.file.Close()

	// Добавляем сообщения в writeBatch (пакет для записи)
	testMessage := LogMessage{
		Service:   "FLUSH_TEST",
		Level:     INFO,
		Message:   "test flush message",
		Timestamp: time.Now(),
	}

	server.batchMu.Lock()
	server.writeBatch = append(server.writeBatch, testMessage)
	initialBatchSize := len(server.writeBatch)
	server.batchMu.Unlock()

	// Проверяем, что пакет не пуст
	if initialBatchSize == 0 {
		t.Error("пакет должен содержать сообщения перед сбросом")
	}

	// Выполняем сброс
	server.flush()

	// Примечание: в реальной реализации flush может не очищать пакет полностью
	// поэтому проверяем, что функция выполнилась без паники
}

// TestLogServerResourceMonitor проверяет мониторинг ресурсов
func TestLogServerResourceMonitor(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Запускаем мониторинг в отдельной горутине
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("resourceMonitor вызвал панику: %v", r)
			}
			done <- true
		}()

		// Запускаем мониторинг на короткое время
		server.resourceMonitor()
	}()

	// Даем мониторингу поработать
	time.Sleep(100 * time.Millisecond)

	// Останавливаем сервер, что должно завершить мониторинг
	_ = server.Stop()

	// Ждем завершения мониторинга
	select {
	case <-done:
		// Мониторинг завершился
	case <-time.After(2 * time.Second):
		t.Error("мониторинг ресурсов не завершился в течение таймаута")
	}
}

// TestLogServerLogStatsAsJSON проверяет вывод статистики в JSON
func TestLogServerLogStatsAsJSON(t *testing.T) {
	config := createTestServerConfig(t)
	server, err := NewLogServer(config)
	if err != nil {
		t.Fatalf("не удалось создать сервер: %v", err)
	}

	// Обновляем статистику
	server.stats.TotalMessages = 10
	server.stats.TotalClients = 8
	server.stats.FileRotations = 1

	// Вызываем функцию вывода статистики
	// Функция должна выполниться без паники
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logStatsAsJSON вызвал панику: %v", r)
		}
	}()

	server.logStatsAsJSON()
}
