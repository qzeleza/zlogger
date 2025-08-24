// logger.go - Основная обёртка для клиентских приложений
package logger

import (
	"fmt"
	"net"
	"os"
	"time"
)

var _ API = (*Logger)(nil) // compile-time guarantee

// Logger основная структура логгера для клиентских приложений
type Logger struct {
	client LogClientInterface
	server *LogServer // Ссылка на сервер для финального flush
}

// New создает новый экземпляр логгера для клиентского приложения
func New(config *LoggingConfig, services []string) (*Logger, error) {
	if config == nil {
		return nil, fmt.Errorf("конфигурация не может быть nil")
	}

	// 1. Запускаем демон логгера
	loggerServer, err := NewLogServer(config)
	if err != nil {
		return nil, err
	}

	if err := loggerServer.Start(); err != nil {
		return nil, err
	}

	// Ждем готовности сокета (до 5 секунд)
	if err := waitForSocket(config.SocketPath, 5*time.Second); err != nil {
		return nil, err
	}

	// Дополнительная задержка для готовности обработчика соединений
	time.Sleep(50 * time.Millisecond)

	// Добавляем переданные сервисы к конфигурационным
	if len(services) > 0 {
		allServices := make(map[string]bool)

		// Добавляем существующие сервисы
		for _, service := range config.Services {
			allServices[service] = true
		}

		// Добавляем новые сервисы
		for _, service := range services {
			allServices[service] = true
		}

		// Обновляем список сервисов
		config.Services = make([]string, 0, len(allServices))
		for service := range allServices {
			config.Services = append(config.Services, service)
		}
	}

	// Создаем клиент
	client, err := NewLogClient(config)
	if err != nil {
		return nil, err
	}

	return &Logger{
		client: client,
		server: loggerServer, // Сохраняем ссылку на сервер
	}, nil
}

// SetService возвращает логгер для указанного сервиса
func (l *Logger) SetService(service string) *ServiceLogger {
	return l.client.SetService(service)
}

// SetLevel устанавливает локальный уровень логирования
func (l *Logger) SetLevel(level LogLevel) {
	l.client.SetLevel(level)
}

// SetServerLevel устанавливает уровень логирования на сервере
func (l *Logger) SetServerLevel(level LogLevel) error {
	return l.client.SetServerLevel(level)
}

// GetLogFile возвращает путь к файлу лога
func (l *Logger) GetLogFile() string {
	return l.client.GetLogFile()
}

// UpdateConfig обновляет конфигурацию логгера
func (l *Logger) UpdateConfig(config *LoggingConfig) error {
	return l.client.UpdateConfig(config)
}

// LogPanic обработчик паники с логированием
func (l *Logger) LogPanic() {
	l.client.LogPanic()
}

// GetLogEntries получает записи из лога с фильтрацией
func (l *Logger) GetLogEntries(filter FilterOptions) ([]LogEntry, error) {
	return l.client.GetLogEntries(filter)
}

// Ping проверяет соединение с сервером
func (l *Logger) Ping() error {
	return l.client.Ping()
}

// Close закрывает логгер
func (l *Logger) Close() error {
	// Принудительно сбрасываем буфер перед закрытием
	if l.server != nil {
		l.server.Flush()
	}

	return l.client.Close()
}

// Методы для MAIN сервиса
func (l *Logger) Debug(message string, args ...interface{}) error {
	if len(args) > 0 {
		return l.client.Debugf(message, args...)
	}
	return l.client.Debug(message)
}

func (l *Logger) Info(message string, args ...interface{}) error {
	if len(args) > 0 {
		return l.client.Infof(message, args...)
	}
	return l.client.Info(message)
}

func (l *Logger) Warn(message string, args ...interface{}) error {
	if len(args) > 0 {
		return l.client.Warnf(message, args...)
	}
	return l.client.Warn(message)
}

func (l *Logger) Error(message string, args ...interface{}) error {
	if len(args) > 0 {
		return l.client.Errorf(message, args...)
	}
	return l.client.Error(message)
}

func (l *Logger) Fatal(message string, args ...interface{}) error {
	if len(args) > 0 {
		return l.client.Fatalf(message, args...)
	}
	return l.client.Fatal(message)
}

func (l *Logger) Panic(message string, args ...interface{}) error {
	if len(args) > 0 {
		return l.client.Panicf(message, args...)
	}
	return l.client.Panic(message)
}

// Форматированные методы для MAIN сервиса
func (l *Logger) Debugf(format string, args ...interface{}) error {
	return l.client.Debugf(format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) error {
	return l.client.Infof(format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) error {
	return l.client.Warnf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) error {
	return l.client.Errorf(format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) error {
	return l.client.Fatalf(format, args...)
}

func (l *Logger) Panicf(format string, args ...interface{}) error {
	return l.client.Panicf(format, args...)
}

// waitForSocket ждет готовности unix сокета с таймаутом
func waitForSocket(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Проверяем существование файла сокета
		if _, err := os.Stat(socketPath); err == nil {
			// Файл существует, пробуем подключиться и сразу отправить тестовое сообщение
			conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
			if err == nil {
				// Пробуем отправить ping
				_, writeErr := conn.Write([]byte("test\n"))
				conn.Close()
				if writeErr == nil {
					return nil // Сокет готов к приему данных
				}
			}
		}

		// Ждем 20мс перед следующей попыткой
		time.Sleep(20 * time.Millisecond)
	}

	return fmt.Errorf("таймаут ожидания готовности сокета %s", socketPath)
}
