package logger

import (
	"fmt"
	"os"
	"time"
)

// ServiceLogger логгер для конкретного сервиса
type ServiceLogger struct {
	client  LogClientInterface
	service string
}

// Ensure ServiceLogger implements logger.API
// var _ API = (*ServiceLogger)(nil)

// SetService возвращает текущий ServiceLogger, так как сервис уже задан
func (s *ServiceLogger) SetService(service string) *ServiceLogger {
	// Переключение сервиса для уже созданного ServiceLogger не поддерживается
	return s
}

// newServiceLogger создает логгер для сервиса
func newServiceLogger(client LogClientInterface, service string) *ServiceLogger {
	return &ServiceLogger{
		client:  client,
		service: service,
	}
}

// Debug записывает debug сообщение с поддержкой различных типов аргументов
func (s *ServiceLogger) Debug(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return s.client.sendMessage(s.service, DEBUG, message, fields)
}

// Info записывает info сообщение с поддержкой различных типов аргументов
func (s *ServiceLogger) Info(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return s.client.sendMessage(s.service, INFO, message, fields)
}

// Warn записывает warning сообщение с поддержкой различных типов аргументов
func (s *ServiceLogger) Warn(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return s.client.sendMessage(s.service, WARN, message, fields)
}

// Error записывает error сообщение с поддержкой различных типов аргументов
func (s *ServiceLogger) Error(args ...interface{}) error {
	// Обрабатываем аргументы и отправляем сообщение
	message, fields := processArgs(args...)
	return s.client.sendMessage(s.service, ERROR, message, fields)
}

// Fatal записывает fatal сообщение и завершает программу
func (s *ServiceLogger) Fatal(args ...interface{}) error {
	// Обрабатываем аргументы
	message, fields := processArgs(args...)

	// Немедленный вывод в stderr, чтобы тесты могли обнаружить "fatal"
	serviceFormatted := fmt.Sprintf("%-5s", s.service)
	levelFormatted := fmt.Sprintf("%-5s", FATAL.String())
	fmt.Fprintf(os.Stderr, "[%s] %s [%s] \"%s\"\n", serviceFormatted, time.Now().Format(DEFAULT_TIME_FORMAT), levelFormatted, message)

	_ = s.client.sendMessage(s.service, FATAL, message, fields)
	fmt.Fprintln(os.Stderr, "fatal")
	os.Exit(1)
	return nil
}

// Panic записывает panic сообщение и вызывает панику
func (s *ServiceLogger) Panic(args ...interface{}) error {
	// Обрабатываем аргументы
	message, fields := processArgs(args...)
	_ = s.client.sendMessage(s.service, PANIC, message, fields)
	panic(message)
}
