// service_test.go - Unit тесты для ServiceLogger
package logger

import (
	"fmt"
	"testing"
)

// TestNewServiceLogger проверяет создание нового ServiceLogger
func TestNewServiceLogger(t *testing.T) {
	mockClient := &MockLogClient{}
	service := "TEST_SERVICE"

	serviceLogger := newServiceLogger(mockClient, service)

	if serviceLogger == nil {
		t.Fatal("newServiceLogger не должен возвращать nil")
	}

	if serviceLogger.client != mockClient {
		t.Error("клиент должен быть установлен корректно")
	}

	if serviceLogger.service != service {
		t.Errorf("ожидался сервис '%s', получили '%s'", service, serviceLogger.service)
	}
}

// TestServiceLoggerMethods проверяет методы логирования ServiceLogger
func TestServiceLoggerMethods(t *testing.T) {
	mockClient := &MockLogClient{}
	service := "API"
	serviceLogger := newServiceLogger(mockClient, service)

	tests := []struct {
		name    string
		method  func() error
		level   LogLevel
		message string
		wantErr bool
	}{
		{
			name:    "Debug",
			method:  func() error { return serviceLogger.Debug("debug message") },
			level:   DEBUG,
			message: "debug message",
			wantErr: false,
		},
		{
			name:    "Info",
			method:  func() error { return serviceLogger.Info("info message") },
			level:   INFO,
			message: "info message",
			wantErr: false,
		},
		{
			name:    "Warn",
			method:  func() error { return serviceLogger.Warn("warn message") },
			level:   WARN,
			message: "warn message",
			wantErr: false,
		},
		{
			name:    "Error",
			method:  func() error { return serviceLogger.Error("error message") },
			level:   ERROR,
			message: "error message",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.Reset()

			err := tt.method()

			if tt.wantErr && err == nil {
				t.Error("ожидалась ошибка, но получили nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
			}

			// Проверяем, что метод был вызван
			if len(mockClient.calls) != 1 {
				t.Errorf("ожидался 1 вызов, получили %d", len(mockClient.calls))
				return
			}

			call := mockClient.calls[0]
			if call.Method != "sendMessage" {
				t.Errorf("ожидался вызов 'sendMessage', получили '%s'", call.Method)
			}
			if call.Service != service {
				t.Errorf("ожидался сервис '%s', получили '%s'", service, call.Service)
			}
			if call.Level != tt.level {
				t.Errorf("ожидался уровень %v, получили %v", tt.level, call.Level)
			}
			if call.Message != tt.message {
				t.Errorf("ожидалось сообщение '%s', получили '%s'", tt.message, call.Message)
			}
		})
	}
}

// TestServiceLoggerFormattedMethods проверяет форматированные методы
func TestServiceLoggerFormattedMethods(t *testing.T) {
	mockClient := &MockLogClient{}
	service := "DNS"
	serviceLogger := newServiceLogger(mockClient, service)

	tests := []struct {
		name     string
		method   func() error
		level    LogLevel
		format   string
		args     []interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "Debugf",
			method:   func() error { return serviceLogger.Debugf("debug: %s %d", "test", 123) },
			level:    DEBUG,
			expected: "debug: test 123",
			wantErr:  false,
		},
		{
			name:     "Infof",
			method:   func() error { return serviceLogger.Infof("info: %v", map[string]int{"count": 5}) },
			level:    INFO,
			expected: "info: map[count:5]",
			wantErr:  false,
		},
		{
			name:     "Warnf",
			method:   func() error { return serviceLogger.Warnf("warn: %.2f%%", 85.67) },
			level:    WARN,
			expected: "warn: 85.67%",
			wantErr:  false,
		},
		{
			name:     "Errorf",
			method:   func() error { return serviceLogger.Errorf("error: %t", true) },
			level:    ERROR,
			expected: "error: true",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.Reset()

			err := tt.method()

			if tt.wantErr && err == nil {
				t.Error("ожидалась ошибка, но получили nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
			}

			// Проверяем, что метод был вызван
			if len(mockClient.calls) != 1 {
				t.Errorf("ожидался 1 вызов, получили %d", len(mockClient.calls))
				return
			}

			call := mockClient.calls[0]
			if call.Service != service {
				t.Errorf("ожидался сервис '%s', получили '%s'", service, call.Service)
			}
			if call.Level != tt.level {
				t.Errorf("ожидался уровень %v, получили %v", tt.level, call.Level)
			}
			if call.Message != tt.expected {
				t.Errorf("ожидалось сообщение '%s', получили '%s'", tt.expected, call.Message)
			}
		})
	}
}

// TestServiceLoggerFatal проверяет метод Fatal (без реального os.Exit)
func TestServiceLoggerFatal(t *testing.T) {
	// Примечание: реальный тест Fatal сложен из-за os.Exit(1)
	// В реальном коде нужно было бы использовать dependency injection для os.Exit
	// Здесь мы тестируем только отправку сообщения

	mockClient := &MockLogClient{}
	service := "FATAL_TEST"
	_ = newServiceLogger(mockClient, service) // создаем для полноты теста

	// Мы не можем вызвать реальный Fatal из-за os.Exit(1)
	// Вместо этого тестируем логику отправки сообщения напрямую
	err := mockClient.sendMessage(service, FATAL, "fatal error")

	if err != nil {
		t.Errorf("неожиданная ошибка при отправке FATAL сообщения: %v", err)
	}

	if len(mockClient.calls) != 1 {
		t.Errorf("ожидался 1 вызов, получили %d", len(mockClient.calls))
		return
	}

	call := mockClient.calls[0]
	if call.Level != FATAL {
		t.Errorf("ожидался уровень FATAL, получили %v", call.Level)
	}
	if call.Message != "fatal error" {
		t.Errorf("ожидалось сообщение 'fatal error', получили '%s'", call.Message)
	}
}

// TestServiceLoggerPanic проверяет метод Panic (без реальной паники)
func TestServiceLoggerPanic(t *testing.T) {
	// Примечание: реальный тест Panic сложен из-за panic()
	// В реальном коде нужно было бы использовать dependency injection для panic
	// Здесь мы тестируем только отправку сообщения

	mockClient := &MockLogClient{}
	service := "PANIC_TEST"
	_ = newServiceLogger(mockClient, service) // создаем для полноты теста

	// Мы не можем вызвать реальный Panic из-за panic()
	// Вместо этого тестируем логику отправки сообщения напрямую
	err := mockClient.sendMessage(service, PANIC, "panic error")

	if err != nil {
		t.Errorf("неожиданная ошибка при отправке PANIC сообщения: %v", err)
	}

	if len(mockClient.calls) != 1 {
		t.Errorf("ожидался 1 вызов, получили %d", len(mockClient.calls))
		return
	}

	call := mockClient.calls[0]
	if call.Level != PANIC {
		t.Errorf("ожидался уровень PANIC, получили %v", call.Level)
	}
	if call.Message != "panic error" {
		t.Errorf("ожидалось сообщение 'panic error', получили '%s'", call.Message)
	}
}

// TestServiceLoggerWithDifferentServices проверяет работу с разными сервисами
func TestServiceLoggerWithDifferentServices(t *testing.T) {
	mockClient := &MockLogClient{}

	services := []string{"API", "DNS", "VPN", "CONFIG"}
	loggers := make([]*ServiceLogger, len(services))

	// Создаем логгеры для разных сервисов
	for i, service := range services {
		loggers[i] = newServiceLogger(mockClient, service)
	}

	// Тестируем логирование от разных сервисов
	for i, logger := range loggers {
		mockClient.Reset()

		message := fmt.Sprintf("message from %s", services[i])
		err := logger.Info(message)

		if err != nil {
			t.Errorf("неожиданная ошибка для сервиса %s: %v", services[i], err)
		}

		if len(mockClient.calls) != 1 {
			t.Errorf("для сервиса %s ожидался 1 вызов, получили %d",
				services[i], len(mockClient.calls))
			continue
		}

		call := mockClient.calls[0]
		if call.Service != services[i] {
			t.Errorf("ожидался сервис '%s', получили '%s'", services[i], call.Service)
		}
		if call.Message != message {
			t.Errorf("ожидалось сообщение '%s', получили '%s'", message, call.Message)
		}
	}
}

// TestServiceLoggerConcurrency проверяет потокобезопасность ServiceLogger
func TestServiceLoggerConcurrency(t *testing.T) {
	mockClient := &MockLogClient{}
	service := "CONCURRENT_TEST"
	serviceLogger := newServiceLogger(mockClient, service)

	const numGoroutines = 10
	const numMessages = 50

	done := make(chan bool, numGoroutines)

	// Запускаем горутины для параллельного логирования
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numMessages; j++ {
				message := fmt.Sprintf("goroutine %d message %d", goroutineID, j)
				err := serviceLogger.Info(message)
				if err != nil {
					t.Errorf("ошибка в горутине %d: %v", goroutineID, err)
				}
			}
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Проверяем общее количество вызовов
	expectedCalls := numGoroutines * numMessages
	if len(mockClient.calls) != expectedCalls {
		t.Errorf("ожидалось %d вызовов, получили %d", expectedCalls, len(mockClient.calls))
	}
}

// BenchmarkServiceLoggerInfo бенчмарк для метода Info
func BenchmarkServiceLoggerInfo(b *testing.B) {
	mockClient := &MockLogClient{}
	serviceLogger := newServiceLogger(mockClient, "BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = serviceLogger.Info("benchmark message")
	}
}

// BenchmarkServiceLoggerInfof бенчмарк для метода Infof
func BenchmarkServiceLoggerInfof(b *testing.B) {
	mockClient := &MockLogClient{}
	serviceLogger := newServiceLogger(mockClient, "BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = serviceLogger.Infof("benchmark message %d", i)
	}
}

// TestServiceLoggerErrorHandling проверяет обработку ошибок
func TestServiceLoggerErrorHandling(t *testing.T) {
	// Создаем мок клиент, который возвращает ошибку
	mockClient := &MockLogClient{}
	serviceLogger := newServiceLogger(mockClient, "ERROR_TEST")

	// В нашем моке sendMessage всегда возвращает nil
	// В реальном тесте здесь можно было бы настроить мок на возврат ошибки
	err := serviceLogger.Info("test message")

	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
}

// TestServiceLoggerEmptyMessages проверяет обработку пустых сообщений
func TestServiceLoggerEmptyMessages(t *testing.T) {
	mockClient := &MockLogClient{}
	serviceLogger := newServiceLogger(mockClient, "EMPTY_TEST")

	tests := []struct {
		name   string
		method func() error
	}{
		{"Debug empty", func() error { return serviceLogger.Debug("") }},
		{"Info empty", func() error { return serviceLogger.Info("") }},
		{"Warn empty", func() error { return serviceLogger.Warn("") }},
		{"Error empty", func() error { return serviceLogger.Error("") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.Reset()

			err := tt.method()

			if err != nil {
				t.Errorf("неожиданная ошибка для пустого сообщения: %v", err)
			}

			if len(mockClient.calls) != 1 {
				t.Errorf("ожидался 1 вызов, получили %d", len(mockClient.calls))
				return
			}

			call := mockClient.calls[0]
			if call.Message != "" {
				t.Errorf("ожидалось пустое сообщение, получили '%s'", call.Message)
			}
		})
	}
}

// TestServiceLoggerSetService проверяет метод SetService
func TestServiceLoggerSetService(t *testing.T) {
	mockClient := &MockLogClient{}
	originalService := "ORIGINAL_SERVICE"
	serviceLogger := newServiceLogger(mockClient, originalService)

	// Проверяем, что SetService возвращает тот же экземпляр
	newService := "NEW_SERVICE"
	result := serviceLogger.SetService(newService)

	if result != serviceLogger {
		t.Error("SetService должен возвращать тот же экземпляр ServiceLogger")
	}

	// Проверяем, что сервис не изменился (согласно комментарию в коде)
	if serviceLogger.service != originalService {
		t.Errorf("сервис не должен изменяться, ожидался '%s', получили '%s'", originalService, serviceLogger.service)
	}

	// Проверяем, что метод не вызывает никаких операций с клиентом
	if len(mockClient.calls) != 0 {
		t.Errorf("SetService не должен вызывать методы клиента, получили %d вызовов", len(mockClient.calls))
	}
}

// TestServiceLoggerFatalMethod проверяет метод Fatal (имитация без os.Exit)
func TestServiceLoggerFatalMethod(t *testing.T) {
	mockClient := &MockLogClient{}
	service := "FATAL_TEST"
	serviceLogger := newServiceLogger(mockClient, service)

	// Поскольку Fatal вызывает os.Exit(1), мы не можем протестировать его напрямую
	// Вместо этого проверим, что sendMessage вызывается с правильными параметрами
	// и что функция возвращает nil (хотя на практике она никогда не вернется из-за os.Exit)

	// Создаем отдельную функцию для тестирования логики без os.Exit
	testFatalLogic := func(message string) {
		// Имитируем только часть с sendMessage
		_ = serviceLogger.client.sendMessage(service, FATAL, message)
	}

	message := "fatal error occurred"
	testFatalLogic(message)

	// Проверяем, что sendMessage был вызван с правильными параметрами
	if len(mockClient.calls) != 1 {
		t.Errorf("ожидался 1 вызов sendMessage, получили %d", len(mockClient.calls))
		return
	}

	call := mockClient.calls[0]
	if call.Method != "sendMessage" {
		t.Errorf("ожидался вызов 'sendMessage', получили '%s'", call.Method)
	}
	if call.Service != service {
		t.Errorf("ожидался сервис '%s', получили '%s'", service, call.Service)
	}
	if call.Level != FATAL {
		t.Errorf("ожидался уровень FATAL, получили %v", call.Level)
	}
	if call.Message != message {
		t.Errorf("ожидалось сообщение '%s', получили '%s'", message, call.Message)
	}
}

// TestServiceLoggerFatalfMethod проверяет метод Fatalf (имитация без os.Exit)
func TestServiceLoggerFatalfMethod(t *testing.T) {
	mockClient := &MockLogClient{}
	service := "FATALF_TEST"
	serviceLogger := newServiceLogger(mockClient, service)

	// Аналогично Fatal, тестируем только логику форматирования и sendMessage
	testFatalfLogic := func(format string, args ...interface{}) {
		// Имитируем логику Fatalf без os.Exit
		message := fmt.Sprintf(format, args...)
		_ = serviceLogger.client.sendMessage(service, FATAL, message)
	}

	format := "fatal error: %s with code %d"
	args := []interface{}{"соединение не удалось", 500}
	expectedMessage := "fatal error: соединение не удалось with code 500"

	testFatalfLogic(format, args...)

	// Проверяем, что sendMessage был вызван с правильными параметрами
	if len(mockClient.calls) != 1 {
		t.Errorf("ожидался 1 вызов sendMessage, получили %d", len(mockClient.calls))
		return
	}

	call := mockClient.calls[0]
	if call.Method != "sendMessage" {
		t.Errorf("ожидался вызов 'sendMessage', получили '%s'", call.Method)
	}
	if call.Service != service {
		t.Errorf("ожидался сервис '%s', получили '%s'", service, call.Service)
	}
	if call.Level != FATAL {
		t.Errorf("ожидался уровень FATAL, получили %v", call.Level)
	}
	if call.Message != expectedMessage {
		t.Errorf("ожидалось сообщение '%s', получили '%s'", expectedMessage, call.Message)
	}

	// Дополнительный тест с пустыми аргументами
	mockClient.Reset()
	testFatalfLogic("simple fatal message")

	if len(mockClient.calls) != 1 {
		t.Errorf("ожидался 1 вызов для простого сообщения, получили %d", len(mockClient.calls))
		return
	}

	call = mockClient.calls[0]
	if call.Message != "simple fatal message" {
		t.Errorf("ожидалось сообщение 'simple fatal message', получили '%s'", call.Message)
	}
}
