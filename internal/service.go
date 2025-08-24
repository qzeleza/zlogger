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

// Debug записывает debug сообщение
func (s *ServiceLogger) Debug(message string) error {
	return s.client.sendMessage(s.service, DEBUG, message)
}

// Info записывает info сообщение
func (s *ServiceLogger) Info(message string) error {
	return s.client.sendMessage(s.service, INFO, message)
}

// Warn записывает warning сообщение
func (s *ServiceLogger) Warn(message string) error {
	return s.client.sendMessage(s.service, WARN, message)
}

// Error записывает error сообщение
func (s *ServiceLogger) Error(message string) error {
	return s.client.sendMessage(s.service, ERROR, message)
}

// Fatal записывает fatal сообщение и завершает программу
func (s *ServiceLogger) Fatal(message string) error {
	// Немедленный вывод в stderr, чтобы тесты могли обнаружить "fatal"
	serviceFormatted := fmt.Sprintf("%-5s", s.service)
	levelFormatted := fmt.Sprintf("%-5s", FATAL.String())
	fmt.Fprintf(os.Stderr, "[%s] %s [%s] \"%s\"\n", serviceFormatted, time.Now().Format(DEFAULT_TIME_FORMAT), levelFormatted, message)

	_ = s.client.sendMessage(s.service, FATAL, message)
	fmt.Fprintln(os.Stderr, "fatal")
	os.Exit(1)
	return nil
}

// Panic записывает panic сообщение и вызывает панику
func (s *ServiceLogger) Panic(message string) error {
	_ = s.client.sendMessage(s.service, PANIC, message)
	panic(message)
}

// Форматированные функции
func (s *ServiceLogger) Debugf(format string, args ...interface{}) error {
	return s.client.sendMessage(s.service, DEBUG, fmt.Sprintf(format, args...))
}

func (s *ServiceLogger) Infof(format string, args ...interface{}) error {
	return s.client.sendMessage(s.service, INFO, fmt.Sprintf(format, args...))
}

func (s *ServiceLogger) Warnf(format string, args ...interface{}) error {
	return s.client.sendMessage(s.service, WARN, fmt.Sprintf(format, args...))
}

func (s *ServiceLogger) Errorf(format string, args ...interface{}) error {
	return s.client.sendMessage(s.service, ERROR, fmt.Sprintf(format, args...))
}

func (s *ServiceLogger) Fatalf(format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	serviceFormatted := fmt.Sprintf("%-5s", s.service)
	levelFormatted := fmt.Sprintf("%-5s", FATAL.String())
	fmt.Fprintf(os.Stderr, "[%s] %s [%s] \"%s\"\n", serviceFormatted, time.Now().Format(DEFAULT_TIME_FORMAT), levelFormatted, message)

	_ = s.client.sendMessage(s.service, FATAL, message)
	fmt.Fprintln(os.Stderr, "fatal")
	os.Exit(1)
	return nil
}

func (s *ServiceLogger) Panicf(format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	_ = s.client.sendMessage(s.service, PANIC, message)
	panic(message)
}
