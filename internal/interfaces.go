// interfaces.go - Интерфейсы для тестирования
package logger

// LogClientInterface интерфейс для клиента логгера
type LogClientInterface interface {
	SetService(service string) *ServiceLogger
	SetLevel(level LogLevel)
	SetServerLevel(level LogLevel) error
	GetLogFile() string
	UpdateConfig(config *LoggingConfig) error
	LogPanic()
	GetLogEntries(filter FilterOptions) ([]LogEntry, error)
	Ping() error
	Close() error

	// Методы логирования для MAIN сервиса
	// Поддерживают различные форматы вызова:
	// - Debug(message string) - простое сообщение
	// - Debug(message string, fields map[string]string) - сообщение с полями в виде карты
	// - Debug(format string, args ...interface{}) - форматированное сообщение
	// - Debug(message string, keyValues ...string) - сообщение с полями в виде пар ключ-значение
	Debug(args ...interface{}) error
	Info(args ...interface{}) error
	Warn(args ...interface{}) error
	Error(args ...interface{}) error
	Fatal(args ...interface{}) error
	Panic(args ...interface{}) error

	// Внутренний метод для отправки сообщений
	sendMessage(service string, level LogLevel, message string, fields map[string]string) error
}
