// integration_test.go - Интеграционные тесты для модуля logger
//go:build logger_integration
// +build logger_integration

package logger

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	conf "kvasdns/internal/config"
)

/**
 * createTestConfig создает тестовую конфигурацию с временным лог-файлом
 * @param t *testing.T - тестовый контекст
 * @param socketPath string - путь к сокету
 * @return *conf.LoggingConfig - конфигурация для тестов
 * @return func() - функция очистки (удаления временного файла)
 */
func createTestConfig(t *testing.T, socketPath string) (*conf.LoggingConfig, func()) {
	// Создаем временный файл для логов
	logFile, err := os.CreateTemp("", "integration_test_*.log")
	if err != nil {
		t.Fatalf("ошибка создания временного лог-файла: %v", err)
	}
	logFile.Close()

	// Функция очистки
	cleanup := func() {
		os.Remove(logFile.Name())
	}

	// Создаем конфигурацию
	config := &conf.LoggingConfig{
		Level:         "DEBUG",
		LogFile:       logFile.Name(),
		SocketPath:    socketPath,
		Services:      []string{"MAIN", "TEST"},
		MaxFileSize:   10,
		MaxFiles:      3,
		BufferSize:    100,
		FlushInterval: 5 * time.Second,
	}

	return config, cleanup
}

// TestLoggerIntegration проверяет интеграцию всех компонентов
func TestLoggerIntegration(t *testing.T) {
	// Пропускаем интеграционные тесты в коротком режиме
	if testing.Short() {
		t.Skip("Пропускаем интеграционные тесты в коротком режиме")
	}

	// Создаем временный сокет
	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "test_logger.sock")

	// Создаем мок сервер
	server := &MockServer{
		socketPath: socketPath,
		messages:   make([]LogMessage, 0),
	}

	// Запускаем сервер в горутине
	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("ошибка запуска мок сервера: %v", err)
		}
	}()

	// Ждем запуска сервера
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	// Создаем тестовую конфигурацию
	config, cleanup := createTestConfig(t, socketPath)
	defer cleanup()

	// Создаем логгер
	logger, err := New(config, []string{"INTEGRATION"})
	if err != nil {
		t.Fatalf("ошибка создания логгера: %v", err)
	}
	defer logger.Close()

	// Тестируем основные методы логирования
	tests := []struct {
		method func() error
		level  LogLevel
	}{
		{func() error { return logger.Debug("integration debug") }, DEBUG},
		{func() error { return logger.Info("integration info") }, INFO},
		{func() error { return logger.Warn("integration warn") }, WARN},
		{func() error { return logger.Error("integration error") }, ERROR},
	}

	for _, tt := range tests {
		err := tt.method()
		if err != nil {
			t.Errorf("ошибка при логировании уровня %v: %v", tt.level, err)
		}
	}

	// Ждем обработки сообщений
	time.Sleep(50 * time.Millisecond)

	// Проверяем, что сообщения дошли до сервера
	server.mu.Lock()
	messageCount := len(server.messages)
	server.mu.Unlock()

	if messageCount != len(tests) {
		t.Errorf("ожидалось %d сообщений, получили %d", len(tests), messageCount)
	}
}

// TestMultipleServices проверяет работу с несколькими сервисами
func TestMultipleServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционные тесты в коротком режиме")
	}

	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "multi_service.sock")

	server := &MockServer{
		socketPath: socketPath,
		messages:   make([]LogMessage, 0),
	}

	go server.Start()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	// Создаем тестовую конфигурацию
	config, cleanup := createTestConfig(t, socketPath)
	defer cleanup()
	config.Level = "INFO"
	config.Services = []string{"MAIN"}

	logger, err := New(config, []string{"API", "DNS", "VPN"})
	if err != nil {
		t.Fatalf("ошибка создания логгера: %v", err)
	}
	defer logger.Close()

	// Тестируем разные сервисы
	services := []string{"API", "DNS", "VPN", "MAIN"}
	for _, service := range services {
		serviceLogger := logger.SetService(service)
		err := serviceLogger.Info(fmt.Sprintf("message from %s", service))
		if err != nil {
			t.Errorf("ошибка логирования для сервиса %s: %v", service, err)
		}
	}

	time.Sleep(50 * time.Millisecond)

	server.mu.Lock()
	messageCount := len(server.messages)
	server.mu.Unlock()

	if messageCount != len(services) {
		t.Errorf("ожидалось %d сообщений, получили %d", len(services), messageCount)
	}
}

// TestConcurrentLogging проверяет параллельное логирование
func TestConcurrentLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционные тесты в коротком режиме")
	}

	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "concurrent.sock")

	server := &MockServer{
		socketPath: socketPath,
		messages:   make([]LogMessage, 0),
	}

	go server.Start()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	config := &conf.LoggingConfig{
		Level:      "INFO",
		SocketPath: socketPath,
		Services:   []string{"MAIN"},
	}

	logger, err := New(config, nil)
	if err != nil {
		t.Fatalf("ошибка создания логгера: %v", err)
	}
	defer logger.Close()

	const numGoroutines = 10
	const messagesPerGoroutine = 20
	var wg sync.WaitGroup

	// Запускаем параллельное логирование
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				message := fmt.Sprintf("goroutine %d message %d", goroutineID, j)
				err := logger.Info(message)
				if err != nil {
					t.Errorf("ошибка в горутине %d: %v", goroutineID, err)
				}
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	server.mu.Lock()
	messageCount := len(server.messages)
	server.mu.Unlock()

	expectedMessages := numGoroutines * messagesPerGoroutine
	if messageCount != expectedMessages {
		t.Errorf("ожидалось %d сообщений, получили %d", expectedMessages, messageCount)
	}
}

// MockServer простой мок сервер для интеграционных тестов
type MockServer struct {
	socketPath string
	listener   net.Listener
	messages   []LogMessage
	mu         sync.Mutex
	stopped    bool
}

// Start запускает мок сервер
func (s *MockServer) Start() error {
	// Удаляем существующий сокет
	os.Remove(s.socketPath)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = listener

	for !s.stopped {
		conn, err := listener.Accept()
		if err != nil {
			if s.stopped {
				return nil
			}
			continue
		}

		go s.handleConnection(conn)
	}

	return nil
}

// Stop останавливает мок сервер
func (s *MockServer) Stop() {
	s.stopped = true
	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketPath)
}

// handleConnection обрабатывает соединение
func (s *MockServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		// Проверяем, не остановлен ли сервер
		if s.stopped {
			return
		}

		var protocolMsg ProtocolMessage
		if err := decoder.Decode(&protocolMsg); err != nil {
			return
		}

		switch protocolMsg.Type {
		case MsgTypeLog:
			// Обрабатываем сообщение лога
			if msgData, ok := protocolMsg.Data.(map[string]interface{}); ok {
				var logMsg LogMessage

				// Простое преобразование из map в структуру
				if service, ok := msgData["service"].(string); ok {
					logMsg.Service = service
				}
				if level, ok := msgData["level"].(float64); ok {
					logMsg.Level = LogLevel(int(level))
				}
				if message, ok := msgData["message"].(string); ok {
					logMsg.Message = message
				}
				if timestamp, ok := msgData["timestamp"].(string); ok {
					if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
						logMsg.Timestamp = t
					}
				}

				s.mu.Lock()
				s.messages = append(s.messages, logMsg)
				s.mu.Unlock()
			}

		case MsgTypePing:
			// Отвечаем на ping
			response := ProtocolMessage{
				Type: MsgTypePong,
				Data: nil,
			}
			encoder.Encode(response)

		case MsgTypeGetEntries:
			// Возвращаем записи
			s.mu.Lock()
			entries := make([]LogEntry, len(s.messages))
			for i, msg := range s.messages {
				entries[i] = LogEntry{
					Service:   msg.Service,
					Level:     msg.Level,
					Message:   msg.Message,
					Timestamp: msg.Timestamp,
					Raw:       fmt.Sprintf("[%s] %s %s", msg.Service, msg.Level.String(), msg.Message),
				}
			}
			s.mu.Unlock()

			response := ProtocolMessage{
				Type: MsgTypeResponse,
				Data: entries,
			}
			encoder.Encode(response)

		case MsgTypeUpdateLevel:
			// Подтверждаем обновление уровня
			response := ProtocolMessage{
				Type: MsgTypeResponse,
				Data: "level updated",
			}
			encoder.Encode(response)
		}
	}
}

// TestReconnection проверяет переподключение при разрыве соединения
func TestReconnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционные тесты в коротком режиме")
	}

	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "reconnect.sock")

	// Запускаем первый сервер
	server1 := &MockServer{
		socketPath: socketPath,
		messages:   make([]LogMessage, 0),
	}

	go server1.Start()
	time.Sleep(100 * time.Millisecond)

	config := &conf.LoggingConfig{
		Level:      "INFO",
		SocketPath: socketPath,
		Services:   []string{"MAIN"},
	}

	logger, err := New(config, nil)
	if err != nil {
		t.Fatalf("ошибка создания логгера: %v", err)
	}
	defer logger.Close()

	// Отправляем сообщение
	err = logger.Info("before reconnection")
	if err != nil {
		t.Errorf("ошибка логирования: %v", err)
	}

	// Останавливаем первый сервер
	server1.Stop()
	time.Sleep(300 * time.Millisecond) // Увеличиваем время ожидания

	// Запускаем второй сервер
	server2 := &MockServer{
		socketPath: socketPath,
		messages:   make([]LogMessage, 0),
	}

	go server2.Start()
	time.Sleep(500 * time.Millisecond) // Увеличиваем время на переподключение
	defer server2.Stop()

	// Сначала отправляем пустое сообщение, чтобы вызвать ошибку и переподключение
	// Это необходимо, так как клиент не знает, что соединение разорвано, пока не попробует отправить сообщение
	// Первая отправка может завершиться ошибкой, но она запустит процесс переподключения
	logger.Info("trigger reconnection")

	// Даем время на переподключение
	time.Sleep(200 * time.Millisecond)

	// Отправляем сообщение после переподключения
	err = logger.Info("after reconnection")
	if err != nil {
		t.Errorf("ошибка логирования после переподключения: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Проверяем, что сообщение дошло до второго сервера
	server2.mu.Lock()
	messageCount := len(server2.messages)
	server2.mu.Unlock()

	if messageCount < 1 {
		t.Error("сообщение не дошло до сервера после переподключения")
	}
}

// TestLoggerWithInvalidSocket проверяет поведение при недоступном сокете
func TestLoggerWithInvalidSocket(t *testing.T) {
	config := &conf.LoggingConfig{
		Level:      "INFO",
		SocketPath: "/nonexistent/path/logger.sock",
		Services:   []string{"MAIN"},
	}

	// Создание логгера должно завершиться ошибкой
	_, err := New(config, nil)
	if err == nil {
		t.Error("ожидалась ошибка при подключении к несуществующему сокету")
	}
}

// TestLoggerPing проверяет ping функциональность
func TestLoggerPing(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционные тесты в коротком режиме")
	}

	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "ping.sock")

	server := &MockServer{
		socketPath: socketPath,
		messages:   make([]LogMessage, 0),
	}

	go server.Start()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	config := &conf.LoggingConfig{
		Level:      "INFO",
		SocketPath: socketPath,
		Services:   []string{"MAIN"},
	}

	logger, err := New(config, nil)
	if err != nil {
		t.Fatalf("ошибка создания логгера: %v", err)
	}
	defer logger.Close()

	// Тестируем ping
	err = logger.Ping()
	if err != nil {
		t.Errorf("ошибка ping: %v", err)
	}
}

// TestGetLogEntries проверяет получение записей лога
func TestGetLogEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционные тесты в коротком режиме")
	}

	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "entries.sock")

	server := &MockServer{
		socketPath: socketPath,
		messages:   make([]LogMessage, 0),
	}

	go server.Start()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	config := &conf.LoggingConfig{
		Level:      "INFO",
		SocketPath: socketPath,
		Services:   []string{"MAIN"},
	}

	logger, err := New(config, nil)
	if err != nil {
		t.Fatalf("ошибка создания логгера: %v", err)
	}
	defer logger.Close()

	// Отправляем несколько сообщений
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err := logger.Info(msg)
		if err != nil {
			t.Errorf("ошибка логирования: %v", err)
		}
	}

	time.Sleep(50 * time.Millisecond)

	// Получаем записи
	filter := FilterOptions{
		Service: "MAIN",
		Limit:   10,
	}

	entries, err := logger.GetLogEntries(filter)
	if err != nil {
		t.Errorf("ошибка получения записей: %v", err)
	}

	if len(entries) != len(messages) {
		t.Errorf("ожидалось %d записей, получили %d", len(messages), len(entries))
	}
}
