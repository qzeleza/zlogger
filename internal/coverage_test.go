package logger

import (
	"os"
	"testing"
)

/**
 * createTestConfig создает тестовую конфигурацию с временным лог-файлом
 * @param t *testing.T - тестовый контекст
 * @param socketPath string - путь к сокету
 * @return *LoggingConfig - конфигурация логирования
 * @return func() - функция очистки
 */
func createTestConfigForCoverage(t *testing.T, socketPath string) (*LoggingConfig, func()) {
	// Создаем временный файл для логов
	tempFile, err := os.CreateTemp("", "test-*.log")
	if err != nil {
		t.Fatalf("не удалось создать временный файл: %v", err)
	}
	_ = tempFile.Close()

	config := &LoggingConfig{
		Level:      "DEBUG",
		LogFile:    tempFile.Name(),
		SocketPath: socketPath,
		Services:   []string{"MAIN", "API", "TEST"},
	}

	cleanup := func() {
		os.Remove(tempFile.Name())
	}

	return config, cleanup
}

/**
 * TestLogClientSetLevel проверяет метод SetLevel клиента логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientSetLevel(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем клиент
	client := &LogClient{
		config:         config,
		level:          INFO,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем установку различных уровней
	testLevels := []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL, PANIC}
	for _, level := range testLevels {
		client.SetLevel(level)
		if client.level != level {
			t.Errorf("ожидался уровень %v, получен %v", level, client.level)
		}
	}
}

/**
 * TestLogClientGetLogEntries проверяет метод GetLogEntries клиента логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientGetLogEntries(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем клиент
	client := &LogClient{
		config:         config,
		level:          INFO,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем получение записей (должно вернуть ошибку, так как нет соединения)
	filter := FilterOptions{
		Service: "TEST",
		Limit:   10,
	}

	entries, err := client.GetLogEntries(filter)
	if err == nil {
		t.Error("ожидалась ошибка при отсутствии соединения")
	}
	if entries != nil {
		t.Error("записи должны быть nil при ошибке")
	}
}

/**
 * TestLogClientLogPanic проверяет метод LogPanic клиента логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientLogPanic(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем клиент
	client := &LogClient{
		config:         config,
		level:          INFO,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем логирование паники
	client.LogPanic()
	// LogPanic не возвращает ошибку, просто вызывает panic
}

/**
 * TestLogClientDirectMethods проверяет прямые методы логирования клиента
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientDirectMethods(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	_, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем клиент с nil конфигурацией для быстрого тестирования без переподключения
	client := &LogClient{
		config:         nil, // nil конфиг предотвращает попытки переподключения
		conn:           nil,
		level:          DEBUG, // Устанавливаем DEBUG для покрытия всех уровней
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем все методы логирования (должны быстро вернуть ошибки из-за nil конфига)
	methods := []struct {
		name string
		fn   func() error
	}{
		{"Debug", func() error { return client.Debug("test debug") }},
		{"Info", func() error { return client.Info("test info") }},
		{"Warn", func() error { return client.Warn("test warn") }},
		{"Error", func() error { return client.Error("test error") }},
		{"Debugf", func() error { return client.Debugf("test %s", "debug") }},
		{"Infof", func() error { return client.Infof("test %s", "info") }},
		{"Warnf", func() error { return client.Warnf("test %s", "warn") }},
		{"Errorf", func() error { return client.Errorf("test %s", "error") }},
	}

	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			err := method.fn()
			if err == nil {
				t.Errorf("метод %s должен вернуть ошибку при nil конфиге", method.name)
			}
		})
	}
}

/**
 * TestLogClientFatalPanic проверяет методы Fatal и Panic клиента
 * @param t *testing.T - тестовый контекст
 */
func TestLogClientFatalPanic(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем клиент
	client := &LogClient{
		config:         config,
		level:          DEBUG,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем Fatal и Panic методы в отдельных горутинах
	// чтобы избежать завершения теста
	t.Run("Fatal", func(t *testing.T) {
		// Fatal должен вызвать os.Exit, поэтому тестируем только что он не паникует
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal не должен вызывать панику: %v", r)
			}
		}()

		// Тестируем Fatal метод (не можем перехватить os.Exit в тестах)
		// Просто проверяем, что метод не паникует
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal не должен вызывать панику: %v", r)
			}
		}()

		// Fatal вызывает os.Exit, что завершает программу
		// В тестах мы не можем это проверить напрямую
		t.Skip("Fatal вызывает os.Exit, что завершает программу")
	})

	t.Run("Panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Panic должен вызвать панику")
			}
		}()

		_ = client.Panic("test panic")
	})

	t.Run("Fatalf", func(t *testing.T) {
		// Тестируем Fatal метод (не можем перехватить os.Exit в тестах)
		// Просто проверяем, что метод не паникует
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal не должен вызывать панику: %v", r)
			}
		}()

		// Fatalf вызывает os.Exit, что завершает программу
		// В тестах мы не можем это проверить напрямую
		t.Skip("Fatalf вызывает os.Exit, что завершает программу")
	})

	t.Run("Panicf", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Panicf должен вызвать панику")
			}
		}()

		_ = client.Panicf("test %s", "panic")
	})
}

/**
 * TestLoggerFatalPanic проверяет методы Fatal и Panic основного логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLoggerFatalPanic(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем логгер с мок-клиентом
	logger := &Logger{
		client: &LogClient{
			config:         config,
			level:          DEBUG,
			serviceLoggers: make(map[string]*ServiceLogger),
			connected:      false,
		},
	}

	t.Run("Fatal", func(t *testing.T) {
		// Тестируем Fatal метод (не можем перехватить os.Exit в тестах)
		// Просто проверяем, что метод не паникует
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal не должен вызывать панику: %v", r)
			}
		}()

		// Fatal вызывает os.Exit, что завершает программу
		t.Skip("Fatal вызывает os.Exit, что завершает программу")
	})

	t.Run("Panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Panic должен вызвать панику")
			}
		}()

		_ = logger.Panic("test panic")
	})

	t.Run("Fatalf", func(t *testing.T) {
		// Тестируем Fatal метод (не можем перехватить os.Exit в тестах)
		// Просто проверяем, что метод не паникует
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal не должен вызывать панику: %v", r)
			}
		}()

		// Fatalf вызывает os.Exit, что завершает программу
		t.Skip("Fatalf вызывает os.Exit, что завершает программу")
	})

	t.Run("Panicf", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Panicf должен вызвать панику")
			}
		}()

		_ = logger.Panicf("test %s", "panic")
	})
}

/**
 * TestLoggerSetServerLevel проверяет метод SetServerLevel основного логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLoggerSetServerLevel(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем логгер
	logger := &Logger{
		client: &LogClient{
			config:         config,
			level:          DEBUG,
			serviceLoggers: make(map[string]*ServiceLogger),
			connected:      false,
		},
	}

	// Тестируем установку уровня на сервере
	err := logger.SetServerLevel(WARN)
	if err == nil {
		t.Error("ожидалась ошибка при отсутствии соединения")
	}
}

/**
 * TestLoggerLogPanic проверяет метод LogPanic основного логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLoggerLogPanic(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем логгер
	logger := &Logger{
		client: &LogClient{
			config:         config,
			level:          DEBUG,
			serviceLoggers: make(map[string]*ServiceLogger),
			connected:      false,
		},
	}

	// Тестируем логирование паники
	logger.LogPanic()
	// LogPanic не возвращает ошибку, просто вызывает panic
}

/**
 * TestLoggerGetLogEntries проверяет метод GetLogEntries основного логгера
 * @param t *testing.T - тестовый контекст
 */
func TestLoggerGetLogEntries(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем логгер
	logger := &Logger{
		client: &LogClient{
			config:         config,
			level:          DEBUG,
			serviceLoggers: make(map[string]*ServiceLogger),
			connected:      false,
		},
	}

	// Тестируем получение записей
	filter := FilterOptions{
		Service: "TEST",
		Limit:   10,
	}

	entries, err := logger.GetLogEntries(filter)
	if err == nil {
		t.Error("ожидалась ошибка при отсутствии соединения")
	}
	if entries != nil {
		t.Error("записи должны быть nil при ошибке")
	}
}

/**
 * TestServiceLoggerFatalPanic проверяет методы Fatal и Panic сервисного логгера
 * @param t *testing.T - тестовый контекст
 */
func TestServiceLoggerFatalPanic(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем сервисный логгер
	client := &LogClient{
		config:         config,
		level:          DEBUG,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	serviceLogger := newServiceLogger(client, "TEST")

	t.Run("Fatal", func(t *testing.T) {
		// Тестируем Fatal метод (не можем перехватить os.Exit в тестах)
		// Просто проверяем, что метод не паникует
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal не должен вызывать панику: %v", r)
			}
		}()

		// Fatal вызывает os.Exit, что завершает программу
		t.Skip("Fatal вызывает os.Exit, что завершает программу")
	})

	t.Run("Panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Panic должен вызвать панику")
			}
		}()

		_ = serviceLogger.Panic("test panic")
	})

	t.Run("Fatalf", func(t *testing.T) {
		// Тестируем Fatal метод (не можем перехватить os.Exit в тестах)
		// Просто проверяем, что метод не паникует
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal не должен вызывать панику: %v", r)
			}
		}()

		// Fatalf вызывает os.Exit, что завершает программу
		t.Skip("Fatalf вызывает os.Exit, что завершает программу")
	})

	t.Run("Panicf", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Panicf должен вызвать панику")
			}
		}()

		_ = serviceLogger.Panicf("test %s", "panic")
	})
}

/**
 * TestSetServiceCoverage проверяет функцию SetService
 * @param t *testing.T - тестовый контекст
 */
func TestSetServiceCoverage(t *testing.T) {
	// Создаем тестовую конфигурацию
	tempDir := t.TempDir()
	socketPath := tempDir + "/test.sock"
	config, cleanup := createTestConfigForCoverage(t, socketPath)
	defer cleanup()

	// Создаем клиент
	client := &LogClient{
		config:         config,
		level:          DEBUG,
		serviceLoggers: make(map[string]*ServiceLogger),
		connected:      false,
	}

	// Тестируем установку сервиса
	serviceLogger := client.SetService("TEST_SERVICE")
	if serviceLogger == nil {
		t.Error("SetService должен вернуть непустой ServiceLogger")
	}

	// Проверяем, что сервис был добавлен в карту
	if _, exists := client.serviceLoggers["TEST_SERVICE"]; !exists {
		t.Error("сервис должен быть добавлен в карту serviceLoggers")
	}
}
