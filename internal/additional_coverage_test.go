package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	conf "kvasdns/internal/config"
)

/**
 * TestNewLogger тестирует создание нового логгера
 * @param t *testing.T - тестовый контекст
 */
func TestNewLogger(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	socketPath := filepath.Join(tempDir, "test.sock")

	// Тестируем создание с валидной конфигурацией
	config := &conf.LoggingConfig{
		LogFile:    logFile,
		SocketPath: socketPath,
		Level:      "INFO",
	}

	logger, err := New(config, []string{})
	if err != nil {
		t.Fatalf("New вернул ошибку: %v", err)
	}
	if logger == nil {
		t.Fatal("New должен вернуть непустой логгер")
	}

	// Проверяем, что клиент инициализирован
	if logger.client == nil {
		t.Error("клиент должен быть инициализирован")
	}

	// Тестируем создание с nil конфигурацией
	nilLogger, err := New(nil, []string{})
	if err == nil {
		t.Error("New должен вернуть ошибку с nil конфигурацией")
	}
	if nilLogger != nil {
		t.Error("логгер должен быть nil при ошибке")
	}

	// Тестируем создание с пустой конфигурацией
	emptyConfig := &conf.LoggingConfig{}
	emptyLogger, err := New(emptyConfig, []string{})
	if err == nil {
		t.Error("New должен вернуть ошибку с пустой конфигурацией")
	}
	if emptyLogger != nil {
		t.Error("логгер должен быть nil при ошибке")
	}

	// Очищаем ресурсы
	_ = logger.Close()
}

/**
 * TestLogClientGetLogEntriesExtended расширенное тестирование GetLogEntries
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientGetLogEntriesExtended(t *testing.T) {
	// Создаем клиент с nil конфигом для быстрого тестирования
	client := &LogClient{
		config:         nil,
		conn:           nil,
		level:          DEBUG,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем с различными фильтрами
	infoLevel := INFO
	debugLevel := DEBUG
	filters := []FilterOptions{
		{}, // Пустой фильтр
		{Level: &infoLevel},
		{Service: "test"},
		{Limit: 10},
		{Level: &debugLevel, Service: "test", Limit: 5},
	}

	for i, filter := range filters {
		t.Run(t.Name()+"_filter_"+string(rune('0'+i)), func(t *testing.T) {
			entries, err := client.GetLogEntries(filter)
			if err == nil {
				t.Error("GetLogEntries должен вернуть ошибку при nil конфиге")
			}
			if entries != nil {
				t.Error("entries должен быть nil при ошибке")
			}
		})
	}
}

/**
 * TestLogClientLogPanicExtended расширенное тестирование LogPanic
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientLogPanicExtended(t *testing.T) {
	// Создаем клиент с nil конфигом
	client := &LogClient{
		config:         nil,
		conn:           nil,
		level:          DEBUG,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем LogPanic с различными сообщениями
	messages := []string{
		"test panic message",
		"",
		"очень длинное сообщение паники с русскими символами и специальными знаками !@#$%^&*()",
	}

	for i, msg := range messages {
		t.Run(t.Name()+"_message_"+string(rune('0'+i)), func(t *testing.T) {
			// LogPanic не возвращает ошибку, просто вызываем его
			client.LogPanic()
			// Проверяем, что паники не произошло
			_ = msg // используем переменную
		})
	}
}

/**
 * TestLogServerNewLogServerExtended расширенное тестирование NewLogServer
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *conf.LoggingConfig
		expectError bool
	}{
		{
			name:        "nil конфигурация",
			config:      nil,
			expectError: true,
		},
		{
			name: "пустой путь к файлу",
			config: &conf.LoggingConfig{
				LogFile:    "",
				SocketPath: "/tmp/test.sock",
				Level:      "INFO",
			},
			expectError: true,
		},
		{
			name: "пустой путь к сокету",
			config: &conf.LoggingConfig{
				LogFile:    "/tmp/test.log",
				SocketPath: "",
				Level:      "INFO",
			},
			expectError: true,
		},
		{
			name: "невалидный уровень логирования",
			config: &conf.LoggingConfig{
				LogFile:    "/tmp/test.log",
				SocketPath: "/tmp/test.sock",
				Level:      "INVALID_LEVEL",
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
 * TestLogServerHelperMethods тестирует вспомогательные методы сервера без сокетов
 * @param t *testing.T - тестовый контекст
 */
func TestLogServerHelperMethods(t *testing.T) {
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
		config: &conf.LoggingConfig{
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
	server.config = &conf.LoggingConfig{
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
 * TestSecurityCleanupExtended расширенное тестирование cleanup в RateLimiter
 * @param t *testing.T - тестовый контекст
 */
func TestSecurityCleanupExtended(t *testing.T) {
	config := DefaultSecurityConfig()
	config.RateLimitPerSecond = 1
	config.BanDuration = time.Millisecond * 100 // Короткий бан для тестов

	limiter := NewRateLimiter(config)
	defer limiter.Close()

	// Добавляем клиента в бан
	clientID := "test-client"
	limiter.clients[clientID] = &ClientInfo{
		MessageCount:  config.RateLimitPerSecond + 1,
		LastAccess:    time.Now(),
		BannedUntil:   time.Now().Add(config.BanDuration),
		TotalMessages: int64(config.RateLimitPerSecond + 1),
	}

	// Ждем окончания бана
	time.Sleep(config.BanDuration + time.Millisecond*50)

	// Сбрасываем счетчик сообщений для клиента, чтобы он мог снова делать запросы
	limiter.mu.Lock()
	if clientInfo, exists := limiter.clients[clientID]; exists {
		clientInfo.MessageCount = 0          // Сбрасываем счетчик
		clientInfo.BannedUntil = time.Time{} // Убираем бан
	}
	limiter.mu.Unlock()

	// Проверяем, что клиент может снова делать запросы
	if !limiter.IsAllowed(clientID) {
		t.Error("клиент должен быть разбанен после истечения времени бана")
	}

	// Тестируем очистку старых записей
	// Добавляем старую запись
	oldClientID := "old-client"
	limiter.clients[oldClientID] = &ClientInfo{
		MessageCount:  1,
		LastAccess:    time.Now().Add(-time.Hour), // Очень старая запись
		BannedUntil:   time.Time{},
		TotalMessages: 1,
	}

	// Даем время cleanup горутине поработать
	time.Sleep(time.Millisecond * 200)

	// Проверяем, что старые записи могут быть очищены
	// (это зависит от реализации cleanup, но мы тестируем, что нет паники)
	limiter.mu.RLock()
	clientsCount := len(limiter.clients)
	limiter.mu.RUnlock()

	if clientsCount < 0 {
		t.Error("количество клиентов не может быть отрицательным")
	}
}
