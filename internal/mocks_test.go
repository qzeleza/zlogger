// mocks_test.go - Моки для тестирования модуля logger
package logger

import (
	"sync"
	"time"
)

// MockError простая реализация ошибки для тестов
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// MockLogClient мок-реализация клиента логгера для тестов
type MockLogClient struct {
	mu                 sync.Mutex
	messages           []LogMessage
	serviceLoggers     map[string]*ServiceLogger
	level              LogLevel
	config             *LoggingConfig
	connected          bool
	failConnect        bool
	failSend           bool
	failSetServerLevel bool
	failGetLogEntries  bool
	failPing           bool
	failClose          bool
	failGetLogFile     bool
	failUpdateConfig   bool
	customSendMessage  func(service string, level LogLevel, message string, fields map[string]string) error
	// Добавляем недостающие поля
	calls      []MockCall
	closed     bool
	logFile    string
	logEntries []LogEntry
	pingError  error
}

// Проверка, что MockLogClient реализует интерфейс LogClientInterface
var _ LogClientInterface = (*MockLogClient)(nil)

// MockCall структура для отслеживания вызовов методов
type MockCall struct {
	Method  string
	Service string
	Level   LogLevel
	Message string
	Args    []interface{}
	Fields  map[string]string
}

// Reset сбрасывает состояние мока
func (m *MockLogClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = nil
	m.closed = false
	m.connected = true
}

// SetService возвращает логгер для указанного сервиса (мок)
func (m *MockLogClient) SetService(service string) *ServiceLogger {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.serviceLoggers == nil {
		m.serviceLoggers = make(map[string]*ServiceLogger)
	}

	if logger, exists := m.serviceLoggers[service]; exists {
		return logger
	}

	logger := &ServiceLogger{
		client:  m,
		service: service,
	}
	m.serviceLoggers[service] = logger

	return logger
}

// SetLevel устанавливает уровень логирования (мок)
func (m *MockLogClient) SetLevel(level LogLevel) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.level = level
	m.calls = append(m.calls, MockCall{
		Method: "SetLevel",
		Level:  level,
	})
}

// SetServerLevel устанавливает уровень на сервере (мок)
func (m *MockLogClient) SetServerLevel(level LogLevel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, MockCall{
		Method: "SetServerLevel",
		Level:  level,
	})
	return nil
}

// GetLogFile возвращает путь к файлу лога (мок)
func (m *MockLogClient) GetLogFile() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.logFile
}

// UpdateConfig обновляет конфигурацию (мок)
func (m *MockLogClient) UpdateConfig(config *LoggingConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
	m.calls = append(m.calls, MockCall{
		Method: "UpdateConfig",
	})
	return nil
}

// LogPanic обработчик паники (мок)
func (m *MockLogClient) LogPanic() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, MockCall{
		Method: "LogPanic",
	})
}

// GetLogEntries получает записи лога (мок)
func (m *MockLogClient) GetLogEntries(filter FilterOptions) ([]LogEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, MockCall{
		Method: "GetLogEntries",
	})

	return m.logEntries, nil
}

// Ping проверяет соединение (мок)
func (m *MockLogClient) Ping() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, MockCall{
		Method: "Ping",
	})

	return m.pingError
}

// Close закрывает соединение (мок)
func (m *MockLogClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	m.calls = append(m.calls, MockCall{
		Method: "Close",
	})
	return nil
}

// sendMessage отправляет сообщение (мок)
func (m *MockLogClient) sendMessage(service string, level LogLevel, message string, fields map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, MockCall{
		Method:  "sendMessage", // Добавляем название метода
		Service: service,
		Level:   level,
		Message: message,
		Fields:  fields,
	})

	// Используем пользовательскую функцию, если она задана
	if m.customSendMessage != nil {
		return m.customSendMessage(service, level, message, fields)
	}
	return nil
}

// Методы логирования для MAIN сервиса (моки)
func (m *MockLogClient) Debug(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return m.sendMessage("MAIN", DEBUG, message, fields)
}

func (m *MockLogClient) Info(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return m.sendMessage("MAIN", INFO, message, fields)
}

func (m *MockLogClient) Warn(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return m.sendMessage("MAIN", WARN, message, fields)
}

func (m *MockLogClient) Error(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return m.sendMessage("MAIN", ERROR, message, fields)
}

func (m *MockLogClient) Fatal(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return m.sendMessage("MAIN", FATAL, message, fields)
}

func (m *MockLogClient) Panic(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return m.sendMessage("MAIN", PANIC, message, fields)
}

// Комментарий: Устаревшие форматированные методы и методы с суффиксом WithFields удалены.
// Теперь все функции логирования используют универсальный интерфейс с вариативными аргументами.

// MockConn мок сетевого соединения
type MockConn struct {
	closed    bool
	readData  []byte
	writeData []byte
	readPos   int
	mu        sync.Mutex
}

// Read читает данные (мок)
func (c *MockConn) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, &MockError{message: "соединение закрыто"}
	}

	if c.readPos >= len(c.readData) {
		return 0, &MockError{message: "EOF"}
	}

	n := copy(b, c.readData[c.readPos:])
	c.readPos += n
	return n, nil
}

// Write записывает данные (мок)
func (c *MockConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, &MockError{message: "соединение закрыто"}
	}

	c.writeData = append(c.writeData, b...)
	return len(b), nil
}

// Close закрывает соединение (мок)
func (c *MockConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	return nil
}

// LocalAddr возвращает локальный адрес (мок)
func (c *MockConn) LocalAddr() interface{} {
	return &MockAddr{network: "unix", address: "/tmp/mock.sock"}
}

// RemoteAddr возвращает удаленный адрес (мок)
func (c *MockConn) RemoteAddr() interface{} {
	return &MockAddr{network: "unix", address: "/tmp/mock.sock"}
}

// SetDeadline устанавливает дедлайн (мок)
func (c *MockConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline устанавливает дедлайн чтения (мок)
func (c *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline устанавливает дедлайн записи (мок)
func (c *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// MockAddr мок адреса
type MockAddr struct {
	network string
	address string
}

// Network возвращает тип сети (мок)
func (a *MockAddr) Network() string {
	return a.network
}

// String возвращает строковое представление адреса (мок)
func (a *MockAddr) String() string {
	return a.address
}

// MockConfig создает мок конфигурации для тестов
func MockConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:      "INFO",
		SocketPath: "/tmp/test_logger.sock",
		Services:   []string{"MAIN", "TEST"},
	}
}

// MockConfigWithLevel создает мок конфигурации с указанным уровнем
func MockConfigWithLevel(level string) *LoggingConfig {
	return &LoggingConfig{
		Level:      level,
		SocketPath: "/tmp/test_logger.sock",
		Services:   []string{"MAIN", "TEST"},
	}
}

// MockConfigWithServices создает мок конфигурации с указанными сервисами
func MockConfigWithServices(services []string) *LoggingConfig {
	return &LoggingConfig{
		Level:      "INFO",
		SocketPath: "/tmp/test_logger.sock",
		Services:   services,
	}
}

// // Функция processArgs перемещена в utils.go
