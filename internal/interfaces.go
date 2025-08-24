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
	Debug(message string) error
	Info(message string) error
	Warn(message string) error
	Error(message string) error
	Fatal(message string) error
	Panic(message string) error

	// Форматированные методы
	Debugf(format string, args ...interface{}) error
	Infof(format string, args ...interface{}) error
	Warnf(format string, args ...interface{}) error
	Errorf(format string, args ...interface{}) error
	Fatalf(format string, args ...interface{}) error
	Panicf(format string, args ...interface{}) error

	// Внутренний метод для отправки сообщений
	sendMessage(service string, level LogLevel, message string) error
}
