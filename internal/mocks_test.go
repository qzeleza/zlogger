// mocks_test.go - Моки для тестирования модуля logger
package logger

import (
	"sync"
	"time"

	conf "kvasdns/internal/config"
)

// MockError простая реализация ошибки для тестов
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// MockLogClient мок клиента логгера для unit тестов
type MockLogClient struct {
	mu                sync.Mutex
	calls             []MockCall
	level             LogLevel
	closed            bool
	logFile           string
	pingError         error
	logEntries        []LogEntry
	serviceLoggers    map[string]*ServiceLogger
	config            *conf.LoggingConfig
	connected         bool
	customSendMessage func(service string, level LogLevel, message string) error
}

// MockCall структура для отслеживания вызовов методов
type MockCall struct {
	Method  string
	Service string
	Level   LogLevel
	Message string
	Args    []interface{}
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
func (m *MockLogClient) UpdateConfig(config *conf.LoggingConfig) error {
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
func (m *MockLogClient) sendMessage(service string, level LogLevel, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, MockCall{
		Method:  "sendMessage",
		Service: service,
		Level:   level,
		Message: message,
	})

	// Используем пользовательскую функцию, если она задана
	if m.customSendMessage != nil {
		return m.customSendMessage(service, level, message)
	}
	return nil
}

// Методы логирования для MAIN сервиса (моки)
func (m *MockLogClient) Debug(message string) error {
	return m.sendMessage("MAIN", DEBUG, message)
}

func (m *MockLogClient) Info(message string) error {
	return m.sendMessage("MAIN", INFO, message)
}

func (m *MockLogClient) Warn(message string) error {
	return m.sendMessage("MAIN", WARN, message)
}

func (m *MockLogClient) Error(message string) error {
	return m.sendMessage("MAIN", ERROR, message)
}

func (m *MockLogClient) Fatal(message string) error {
	return m.sendMessage("MAIN", FATAL, message)
}

func (m *MockLogClient) Panic(message string) error {
	return m.sendMessage("MAIN", PANIC, message)
}

// Форматированные методы (моки)
func (m *MockLogClient) Debugf(format string, args ...interface{}) error {
	return m.sendMessage("MAIN", DEBUG, format)
}

func (m *MockLogClient) Infof(format string, args ...interface{}) error {
	return m.sendMessage("MAIN", INFO, format)
}

func (m *MockLogClient) Warnf(format string, args ...interface{}) error {
	return m.sendMessage("MAIN", WARN, format)
}

func (m *MockLogClient) Errorf(format string, args ...interface{}) error {
	return m.sendMessage("MAIN", ERROR, format)
}

func (m *MockLogClient) Fatalf(format string, args ...interface{}) error {
	return m.sendMessage("MAIN", FATAL, format)
}

func (m *MockLogClient) Panicf(format string, args ...interface{}) error {
	return m.sendMessage("MAIN", PANIC, format)
}

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
func MockConfig() *conf.LoggingConfig {
	return &conf.LoggingConfig{
		Level:      "INFO",
		SocketPath: "/tmp/test_logger.sock",
		Services:   []string{"MAIN", "TEST"},
	}
}

// MockConfigWithLevel создает мок конфигурации с указанным уровнем
func MockConfigWithLevel(level string) *conf.LoggingConfig {
	return &conf.LoggingConfig{
		Level:      level,
		SocketPath: "/tmp/test_logger.sock",
		Services:   []string{"MAIN", "TEST"},
	}
}

// MockConfigWithServices создает мок конфигурации с указанными сервисами
func MockConfigWithServices(services []string) *conf.LoggingConfig {
	return &conf.LoggingConfig{
		Level:      "INFO",
		SocketPath: "/tmp/test_logger.sock",
		Services:   services,
	}
}
