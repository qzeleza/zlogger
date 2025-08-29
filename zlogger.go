// Package zlogger - Высокопроизводительная система логирования для embedded систем
// Предоставляет простой API для интеграции в сторонние приложения
package zlogger

import (
	"time"

	logger "github.com/qzeleza/zlogger/internal"
)

// Экспортируемые типы для публичного API
type (
	// Logger основной интерфейс логгера для клиентских приложений
	Logger = logger.Logger

	// ServiceLogger логгер для конкретного сервиса
	ServiceLogger = logger.ServiceLogger

	// LogLevel уровни логирования
	LogLevel = logger.LogLevel

	// Config конфигурация системы логирования
	Config = logger.LoggingConfig

	// LogEntry запись лога для чтения
	LogEntry = logger.LogEntry

	// FilterOptions опции фильтрации логов
	FilterOptions = logger.FilterOptions
)

// Экспортируемые константы уровней логирования
const (
	DEBUG LogLevel = logger.DEBUG // Отладочная информация
	INFO  LogLevel = logger.INFO  // Информационные сообщения
	WARN  LogLevel = logger.WARN  // Предупреждения
	ERROR LogLevel = logger.ERROR // Ошибки
	FATAL LogLevel = logger.FATAL // Критические ошибки
	PANIC LogLevel = logger.PANIC // Паника приложения
)

// New создает новый экземпляр логгера с указанной конфигурацией
//
// Параметры:
//   - config: конфигурация логгера (обязательный)
//   - services: список дополнительных сервисов для логирования (опционально)
//
// Возвращает:
//   - *Logger: экземпляр логгера
//   - error: ошибка инициализации
//
// Пример использования:
//
//	config := &zlogger.Config{
//	    Level:         "info",
//	    LogFile:       "/var/log/myapp.log",
//	    SocketPath:    "/tmp/myapp.sock",
//	    MaxFileSize:   100,
//	    BufferSize:    1000,
//	    FlushInterval: time.Second,
//	}
//
//	log, err := zlogger.New(config, []string{"API", "DB", "CACHE"})
//	if err != nil {
//	    panic(err)
//	}
//	defer log.Close()
func New(config *Config, services ...string) (*Logger, error) {
	var serviceList []string
	if len(services) > 0 {
		serviceList = services
	}
	return logger.New(config, serviceList)
}

// NewConfig создает конфигурацию с настройками по умолчанию
//
// Параметры:
//   - logFile: путь к файлу лога
//   - socketPath: путь к Unix сокету
//
// Возвращает готовую к использованию конфигурацию по умолчанию
func NewConfig(logFile, socketPath string) *Config {
	return &Config{
		Level:            "info",      // Уровень логирования
		LogFile:          logFile,     // Путь к файлу лога
		SocketPath:       socketPath,  // Путь к Unix сокету
		MaxFileSize:      1,           // 1 MB
		BufferSize:       1000,        // 1000 сообщений
		FlushInterval:    time.Second, // 1 секунда
		Services:         []string{},  // Пустой список сервисов
		RestrictServices: false,       // Не ограничивать сервисы
		Dir:              "/tmp",      // Путь к директории логов
		MaxFiles:         3,           // Максимальное количество файлов логов
		MaxSize:          1,           // Максимальный размер файла логов в MB
		MaxAge:           7,           // Максимальный возраст файла логов
		Compress:         true,        // Сжатие файлов логов
		Console:          true,        // Вывод логов в консоль
		MaxBackups:       3,           // Максимальное количество резервных копий
	}
}

// ParseLevel парсит строковый уровень логирования
//
// Параметры:
//   - level: строковое представление уровня ("debug", "info", "warn", "error", "fatal", "panic")
//
// Возвращает:
//   - LogLevel: уровень логирования
//   - error: ошибка парсинга
func ParseLevel(level string) (LogLevel, error) {
	return logger.ParseLevel(level)
}

// Глобальные функции для быстрого логирования без создания экземпляра
// Используют простой вывод в stdout/stderr

// Debug выводит отладочное сообщение с поддержкой различных типов аргументов
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Info выводит информационное сообщение с поддержкой различных типов аргументов
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn выводит предупреждение с поддержкой различных типов аргументов
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error выводит сообщение об ошибке с поддержкой различных типов аргументов
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Fatal выводит критическое сообщение с поддержкой различных типов аргументов
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Panic выводит сообщение паники с поддержкой различных типов аргументов
func Panic(args ...interface{}) {
	logger.Panic(args...)
}
