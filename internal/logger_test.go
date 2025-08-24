// logger_test.go - Unit тесты для основного логгера
package logger

import (
	"testing"
)

// TestNew проверяет создание нового логгера
func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		config   *LoggingConfig
		services []string
		wantErr  bool
	}{
		{
			name:     "создание с nil конфигурацией",
			config:   nil,
			services: []string{"TEST"},
			wantErr:  true, // Ожидаем ошибку подключения к несуществующему серверу
		},
		{
			name: "создание с валидной конфигурацией",
			config: &LoggingConfig{
				Level:      "INFO",
				SocketPath: "/tmp/test_logger.sock",
				Services:   []string{"MAIN"},
			},
			services: []string{"TEST"},
			wantErr:  true, // Ожидаем ошибку подключения к несуществующему серверу
		},
		{
			name: "создание с дублирующимися сервисами",
			config: &LoggingConfig{
				Level:      "DEBUG",
				SocketPath: "/tmp/test_logger.sock",
				Services:   []string{"MAIN", "TEST"},
			},
			services: []string{"TEST", "API"},
			wantErr:  true, // Ожидаем ошибку подключения к несуществующему серверу
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config, tt.services)

			if tt.wantErr {
				if err == nil {
					t.Error("ожидалась ошибка, но получили nil")
				}
				return
			}

			if err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
				return
			}

			if logger == nil {
				t.Error("логгер не должен быть nil")
			} else if logger.client == nil {
				// Проверяем, что клиент создан
				t.Error("клиент логгера не должен быть nil")
			}
		})
	}
}

// TestLoggerMethods проверяет методы логирования
func TestLoggerMethods(t *testing.T) {
	// Создаем мок логгер с мок клиентом
	mockClient := &MockLogClient{}
	logger := &Logger{client: mockClient}

	tests := []struct {
		name    string
		method  func() error
		level   LogLevel
		message string
		wantErr bool
	}{
		{
			name:    "Debug",
			method:  func() error { return logger.Debug("test debug") },
			level:   DEBUG,
			message: "test debug",
			wantErr: false,
		},
		{
			name:    "Info",
			method:  func() error { return logger.Info("test info") },
			level:   INFO,
			message: "test info",
			wantErr: false,
		},
		{
			name:    "Warn",
			method:  func() error { return logger.Warn("test warn") },
			level:   WARN,
			message: "test warn",
			wantErr: false,
		},
		{
			name:    "Error",
			method:  func() error { return logger.Error("test error") },
			level:   ERROR,
			message: "test error",
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
			}
		})
	}
}

// TestLoggerFormattedMethods проверяет форматированные методы логирования
func TestLoggerFormattedMethods(t *testing.T) {
	mockClient := &MockLogClient{}
	logger := &Logger{client: mockClient}

	tests := []struct {
		name     string
		method   func() error
		format   string
		args     []interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "Debugf",
			method:   func() error { return logger.Debugf("debug: %s %d", "test", 123) },
			expected: "debug: test 123",
			wantErr:  false,
		},
		{
			name:     "Infof",
			method:   func() error { return logger.Infof("info: %v", map[string]int{"count": 5}) },
			expected: "info: map[count:5]",
			wantErr:  false,
		},
		{
			name:     "Warnf",
			method:   func() error { return logger.Warnf("warn: %.2f%%", 85.67) },
			expected: "warn: 85.67%",
			wantErr:  false,
		},
		{
			name:     "Errorf",
			method:   func() error { return logger.Errorf("error: %t", true) },
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
		})
	}
}

// TestSetService проверяет получение логгера для сервиса
func TestSetService(t *testing.T) {
	mockClient := &MockLogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}
	logger := &Logger{client: mockClient}

	// Тестируем создание логгера для сервиса
	serviceLogger := logger.SetService("TEST_SERVICE")

	if serviceLogger == nil {
		t.Error("логгер сервиса не должен быть nil")
	}

	// Проверяем кеширование - второй вызов должен вернуть тот же объект
	serviceLogger2 := logger.SetService("TEST_SERVICE")

	if serviceLogger != serviceLogger2 {
		t.Error("логгеры сервисов должны кешироваться")
	}
}

// TestSetLevel проверяет установку уровня логирования
func TestSetLevel(t *testing.T) {
	mockClient := &MockLogClient{}
	logger := &Logger{client: mockClient}

	// Тестируем установку различных уровней
	levels := []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL, PANIC}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			logger.SetLevel(level)

			if mockClient.level != level {
				t.Errorf("ожидался уровень %v, получили %v", level, mockClient.level)
			}
		})
	}
}

// TestClose проверяет закрытие логгера
func TestClose(t *testing.T) {
	mockClient := &MockLogClient{}
	logger := &Logger{client: mockClient}

	err := logger.Close()

	if err != nil {
		t.Errorf("неожиданная ошибка при закрытии: %v", err)
	}

	if !mockClient.closed {
		t.Error("клиент должен быть закрыт")
	}
}

// TestGetLogFile проверяет получение пути к файлу лога
func TestGetLogFile(t *testing.T) {
	mockClient := &MockLogClient{
		logFile: "/tmp/test.log",
	}
	logger := &Logger{client: mockClient}

	logFile := logger.GetLogFile()

	if logFile != "/tmp/test.log" {
		t.Errorf("ожидался путь '/tmp/test.log', получили '%s'", logFile)
	}
}

// TestUpdateConfig проверяет обновление конфигурации
func TestUpdateConfig(t *testing.T) {
	mockClient := &MockLogClient{}
	logger := &Logger{client: mockClient}

	newConfig := &LoggingConfig{
		Level:      "ERROR",
		SocketPath: "/tmp/new_logger.sock",
		Services:   []string{"NEW_SERVICE"},
	}

	err := logger.UpdateConfig(newConfig)

	if err != nil {
		t.Errorf("неожиданная ошибка при обновлении конфигурации: %v", err)
	}
}

// TestPing проверяет проверку соединения
func TestPing(t *testing.T) {
	tests := []struct {
		name      string
		mockError error
		wantErr   bool
	}{
		{
			name:      "успешный ping",
			mockError: nil,
			wantErr:   false,
		},
		{
			name:      "ошибка ping",
			mockError: &MockError{message: "соединение не удалось"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLogClient{
				pingError: tt.mockError,
			}
			logger := &Logger{client: mockClient}

			err := logger.Ping()

			if tt.wantErr && err == nil {
				t.Error("ожидалась ошибка, но получили nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
			}
		})
	}
}
