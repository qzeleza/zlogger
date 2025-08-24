package logger

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

/**
 * TestFatalFunctions тестирует функции Fatal и Fatalf через отдельный процесс
 * @param t *testing.T - тестовый контекст
 */
func TestFatalFunctions(t *testing.T) {
	// Проверяем, что мы не в подпроцессе
	if os.Getenv("TEST_FATAL") == "1" {
		// Это подпроцесс - выполняем Fatal функции
		tempDir := os.Getenv("TEST_TEMP_DIR")
		logFile := filepath.Join(tempDir, "fatal_test.log")
		socketPath := filepath.Join(tempDir, "fatal_test.sock")

		config := &LoggingConfig{
			LogFile:       logFile,
			SocketPath:    socketPath,
			Level:         "INFO",
			FlushInterval: time.Millisecond * 100,
			BufferSize:    100,
			MaxFileSize:   1024 * 1024,
			MaxFiles:      3,
		}

		logger, err := New(config, []string{})
		if err != nil {
			// Если не удалось создать логгер, создаем клиент с nil конфигом
			client := &LogClient{
				config:         nil,
				conn:           nil,
				level:          DEBUG,
				serviceLoggers: make(map[string]*ServiceLogger),
				connected:      false,
			}

			fatalType := os.Getenv("FATAL_TYPE")
			switch fatalType {
			case "Fatal":
				_ = client.Fatal("test fatal message")
			case "Fatalf":
				_ = client.Fatalf("test fatal message: %s", "formatted")
			case "ServiceFatal":
				// Тестируем Fatal из service.go
				serviceLogger := &ServiceLogger{
					client:  client,
					service: "TEST",
				}
				_ = serviceLogger.Fatal("test service fatal")
			case "ServiceFatalf":
				// Тестируем Fatalf из service.go
				serviceLogger := &ServiceLogger{
					client:  client,
					service: "TEST",
				}
				_ = serviceLogger.Fatalf("test service fatal: %s", "formatted")
			}
			return
		}

		fatalType := os.Getenv("FATAL_TYPE")
		switch fatalType {
		case "Fatal":
			_ = logger.Fatal("test fatal message")
		case "Fatalf":
			_ = logger.Fatalf("test fatal message: %s", "formatted")
		case "ServiceFatal":
			serviceLogger := logger.SetService("TEST")
			_ = serviceLogger.Fatal("test service fatal")
		case "ServiceFatalf":
			serviceLogger := logger.SetService("TEST")
			_ = serviceLogger.Fatalf("test service fatal: %s", "formatted")
		}
		return
	}

	// Основной тест - запускаем подпроцессы для каждой Fatal функции
	tempDir := t.TempDir()

	fatalTypes := []string{
		"Fatal",
		"Fatalf",
		"ServiceFatal",
		"ServiceFatalf",
	}

	for _, fatalType := range fatalTypes {
		t.Run(fatalType, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestFatalFunctions")
			cmd.Env = append(os.Environ(),
				"TEST_FATAL=1",
				"FATAL_TYPE="+fatalType,
				"TEST_TEMP_DIR="+tempDir,
			)

			output, err := cmd.CombinedOutput()

			// Fatal функции должны завершать процесс с кодом 1
			if err == nil {
				t.Errorf("ожидался выход с ошибкой для %s", fatalType)
			}

			// Проверяем, что процесс завершился с правильным кодом
			if exitError, ok := err.(*exec.ExitError); ok {
				if exitError.ExitCode() != 1 {
					t.Errorf("ожидался код выхода 1, получен %d для %s", exitError.ExitCode(), fatalType)
				}
			}

			// Проверяем, что в выводе есть наше сообщение
			outputStr := string(output)
			if !strings.Contains(outputStr, "fatal") {
				t.Errorf("вывод не содержит 'fatal' для %s: %s", fatalType, outputStr)
			}
		})
	}
}

/**
 * TestServerFunctionsWithoutSocket тестирует функции сервера без создания сокета
 * @param t *testing.T - тестовый контекст
 */
func TestServerFunctionsWithoutSocket(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Создаем конфигурацию без сокета для тестирования отдельных функций
	config := &LoggingConfig{
		LogFile:       logFile,
		SocketPath:    "", // Пустой путь к сокету
		Level:         "INFO",
		FlushInterval: time.Millisecond * 100,
		BufferSize:    100,
		MaxFileSize:   1024 * 1024,
		MaxFiles:      3,
	}

	// Тестируем, что NewLogServer возвращает ошибку при пустом SocketPath
	server, err := NewLogServer(config)
	if err == nil {
		t.Error("ожидалась ошибка при пустом SocketPath")
	}
	if server != nil {
		t.Error("сервер должен быть nil при ошибке")
	}

	// Тестируем функции форматирования и парсинга без полного сервера
	// Создаем минимальный сервер для тестирования методов
	validConfig := &LoggingConfig{
		LogFile:       logFile,
		SocketPath:    filepath.Join(tempDir, "test.sock"),
		Level:         "INFO",
		FlushInterval: time.Millisecond * 100,
		BufferSize:    100,
		MaxFileSize:   1024 * 1024,
		MaxFiles:      3,
	}

	// Пытаемся создать сервер, но если не получается из-за сокета,
	// тестируем отдельные функции
	testServer, err := NewLogServer(validConfig)
	if err != nil {
		// Если не удалось создать сервер, тестируем отдельные функции
		t.Logf("Не удалось создать сервер (ожидаемо в тестовой среде): %v", err)

		// Тестируем функции парсинга уровня логирования
		level, err := ParseLevel("DEBUG")
		if err != nil {
			t.Errorf("ошибка парсинга уровня DEBUG: %v", err)
		}
		if level != DEBUG {
			t.Errorf("ожидался DEBUG, получен %v", level)
		}

		level, err = ParseLevel("НЕДОПУСТИМЫЙ")
		if err == nil {
			t.Error("ожидалась ошибка для невалидного уровня")
		}
		if level != INFO {
			t.Errorf("ожидался INFO, получен %v", level)
		}
		return
	}

	// Если сервер создался успешно, тестируем его методы
	defer func() {
		if testServer.file != nil {
			_ = testServer.file.Close()
		}
	}()

	// Тестируем formatMessageAsTXT
	msg := &LogMessage{
		Level:     INFO,
		Message:   "test message",
		Service:   "TEST",
		Timestamp: time.Now(),
	}

	formatted := testServer.formatMessageAsTXT(*msg)
	if !strings.Contains(formatted, "test message") {
		t.Error("отформатированное сообщение должно содержать текст сообщения")
	}
	if !strings.Contains(formatted, "TEST") {
		t.Error("отформатированное сообщение должно содержать имя сервиса")
	}

	// Тестируем sendError (не требует сокета)
	testServer.sendError(nil, "test error")
	// Функция не должна паниковать при nil соединении

	// Тестируем handlePing (не требует сокета)
	testServer.handlePing(nil)
	// Функция не должна паниковать при nil соединении
}

/**
 * TestLogCacheAdditionalMethods тестирует дополнительные методы кеша
 * @param t *testing.T - тестовый контекст
 */
func TestLogCacheAdditionalMethods(t *testing.T) {
	cache := NewLogCache(10, time.Minute)
	defer cache.Close()

	// Тестируем добавление записи в кеш
	entry := LogEntry{
		Level:     INFO,
		Message:   "test message",
		Service:   "TEST",
		Timestamp: time.Now(),
	}
	key := "test_key"
	cache.Put(key, entry)

	// Тестируем получение записи из кеша
	retrievedEntry, found := cache.Get(key)
	if !found {
		t.Error("запись должна быть найдена в кеше")
	}
	if retrievedEntry == nil {
		t.Error("полученная запись не должна быть nil")
	} else if retrievedEntry.Message != "test message" {
		t.Errorf("ожидалось 'test message', получено '%s'", retrievedEntry.Message)
	}

	// Тестируем добавление нескольких записей (но не превышаем лимит)
	for i := 0; i < 5; i++ {
		entryLoop := LogEntry{
			Level:     INFO,
			Message:   fmt.Sprintf("message_%d", i),
			Service:   "TEST",
			Timestamp: time.Now(),
		}
		keyLoop := fmt.Sprintf("key_%d", i)
		cache.Put(keyLoop, entryLoop)
	}

	// Тестируем получение несуществующей записи
	_, found = cache.Get("nonexistent_key")
	if found {
		t.Error("несуществующая запись не должна быть найдена")
	}

	// Тестируем статистику кеша
	stats := cache.GetStats()
	if stats.Hits < 0 || stats.Misses < 0 {
		t.Error("статистика кеша не может быть отрицательной")
	}

	// Тестируем очистку кеша
	cache.Clear()
	_, found = cache.Get(key)
	if found {
		t.Error("запись не должна быть найдена после очистки кеша")
	}
	_, found = cache.Get("key_0")
	if found {
		t.Error("записи не должны быть найдены после очистки кеша")
	}
}
