package main

import (
	"fmt"
	"os"
	"time"

	"github.com/qzeleza/zlogger"
)

func main() {
	// Создаем конфигурацию с настройками по умолчанию
	config := zlogger.NewConfig("/tmp/example.log", "/tmp/example.sock")

	// Настраиваем дополнительные параметры
	config.Level = "debug"
	config.MaxFileSize = 50 // 50 MB
	config.BufferSize = 500
	config.FlushInterval = 2 * time.Second

	// Создаем логгер с дополнительными сервисами
	logger, err := zlogger.New(config, "API", "DATABASE", "CACHE", "AUTH")
	if err != nil {
		fmt.Printf("Ошибка создания логгера: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Демонстрация основных возможностей библиотеки
	demonstrateBasicLogging(logger)
	demonstrateServiceLogging(logger)
	demonstrateFormattedLogging(logger)
	demonstrateLogRetrieval(logger)
	demonstrateGlobalFunctions()
}

// demonstrateBasicLogging демонстрирует базовое логирование
func demonstrateBasicLogging(logger *zlogger.Logger) {
	fmt.Println("=== Демонстрация базового логирования ===")

	// Логирование с разными уровнями для основного сервиса (MAIN)
	logger.Debug("Отладочное сообщение - детали работы приложения")
	logger.Info("Информационное сообщение - приложение запущено")
	logger.Warn("Предупреждение - использование устаревшего API")
	logger.Error("Ошибка - не удалось подключиться к базе данных")

	// Примечание: Fatal и Panic завершают программу, поэтому не вызываем их в примере
	fmt.Println("Базовые сообщения записаны в лог")
}

// demonstrateServiceLogging демонстрирует логирование по сервисам
func demonstrateServiceLogging(logger *zlogger.Logger) {
	fmt.Println("\n=== Демонстрация логирования по сервисам ===")

	// Получаем логгеры для разных сервисов
	apiLogger := logger.SetService("API")
	dbLogger := logger.SetService("DATABASE")
	cacheLogger := logger.SetService("CACHE")
	authLogger := logger.SetService("AUTH")

	// Логирование от разных сервисов
	apiLogger.Info("Получен HTTP запрос GET /users")
	dbLogger.Debug("Выполнение SQL запроса: SELECT * FROM users")
	cacheLogger.Info("Кеш обновлен для ключа: users_list")
	authLogger.Warn("Неудачная попытка входа для пользователя: admin")

	apiLogger.Info("HTTP ответ отправлен: 200 OK")
	dbLogger.Info("Запрос выполнен успешно, получено 150 записей")

	fmt.Println("Сообщения от разных сервисов записаны в лог")
}

// demonstrateFormattedLogging демонстрирует форматированное логирование
func demonstrateFormattedLogging(logger *zlogger.Logger) {
	fmt.Println("\n=== Демонстрация форматированного логирования ===")

	// Форматированное логирование для основного сервиса
	userID := 12345
	requestTime := 150
	logger.Infof("Пользователь %d выполнил запрос за %d мс", userID, requestTime)
	logger.Debugf("Детали запроса: метод=%s, путь=%s, IP=%s", "GET", "/api/users", "192.168.1.100")

	// Форматированное логирование для сервисов
	dbLogger := logger.SetService("DATABASE")
	dbLogger.Errorf("Ошибка подключения к БД: %s, попытка %d из %d", "connection timeout", 3, 5)

	cacheLogger := logger.SetService("CACHE")
	cacheLogger.Infof("Статистика кеша: попаданий=%d, промахов=%d, коэффициент=%.2f",
		850, 150, 0.85)

	fmt.Println("Форматированные сообщения записаны в лог")
}

// demonstrateLogRetrieval демонстрирует получение записей из лога
func demonstrateLogRetrieval(logger *zlogger.Logger) {
	fmt.Println("\n=== Демонстрация получения записей из лога ===")

	// Ждем немного, чтобы сообщения записались
	time.Sleep(100 * time.Millisecond)

	// Создаем фильтр для получения записей
	filter := &zlogger.FilterOptions{
		Service: "API",
		Limit:   10,
	}

	// Получаем записи из лога
	entries, err := logger.GetLogEntries(*filter)
	if err != nil {
		fmt.Printf("Ошибка получения записей: %v\n", err)
		return
	}

	fmt.Printf("Найдено %d записей для сервиса API:\n", len(entries))
	for i, entry := range entries {
		fmt.Printf("  %d. [%s] %s: %s\n",
			i+1, entry.Level.String(), entry.Timestamp.Format("15:04:05"), entry.Message)
	}

	// Получаем последние записи всех сервисов
	allFilter := &zlogger.FilterOptions{
		Limit: 5,
	}

	allEntries, err := logger.GetLogEntries(*allFilter)
	if err != nil {
		fmt.Printf("Ошибка получения всех записей: %v\n", err)
		return
	}

	fmt.Printf("\nПоследние %d записей:\n", len(allEntries))
	for i, entry := range allEntries {
		fmt.Printf("  %d. [%s] %s [%s]: %s\n",
			i+1, entry.Service, entry.Timestamp.Format("15:04:05"),
			entry.Level.String(), entry.Message)
	}
}

// demonstrateGlobalFunctions демонстрирует глобальные функции
func demonstrateGlobalFunctions() {
	fmt.Println("\n=== Демонстрация глобальных функций ===")

	// Глобальные функции для быстрого логирования (выводят в stdout/stderr)
	zlogger.Debug("Глобальное отладочное сообщение")
	zlogger.Info("Глобальное информационное сообщение")
	zlogger.Warn("Глобальное предупреждение")
	zlogger.Error("Глобальная ошибка")

	// Форматированные глобальные функции
	zlogger.Info("Глобальное форматированное сообщение: значение=%d", 42)
	zlogger.Debug("Отладка: переменная=%s, статус=%t", "test", true)

	fmt.Println("Глобальные функции продемонстрированы")
}

// Дополнительные примеры использования в комментариях:
/*
// Пример обработки паники
func examplePanicHandling(logger *zlogger.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Восстановлено после паники: %v", r)
		}
	}()

	// Код, который может вызвать панику
	panic("тестовая паника")
}

// Пример изменения уровня логирования
func exampleLevelChange(logger *zlogger.Logger) {
	// Изменяем локальный уровень
	logger.SetLevel(zlogger.ERROR)

	// Изменяем уровень на сервере
	err := logger.SetServerLevel(zlogger.WARN)
	if err != nil {
		fmt.Printf("Ошибка изменения уровня: %v\n", err)
	}
}

// Пример проверки соединения
func exampleHealthCheck(logger *zlogger.Logger) {
	err := logger.Ping()
	if err != nil {
		fmt.Printf("Сервер логгера недоступен: %v\n", err)
	} else {
		fmt.Println("Соединение с сервером логгера активно")
	}
}

// Пример фильтрации по времени
func exampleTimeFiltering(logger *zlogger.Logger) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	filter := &zlogger.FilterOptions{
		StartTime: &startTime,
		EndTime:   &endTime,
		Level:     &[]zlogger.LogLevel{zlogger.ERROR}[0],
		Limit:     100,
	}

	entries, err := logger.GetLogEntries(*filter)
	if err != nil {
		fmt.Printf("Ошибка фильтрации: %v\n", err)
		return
	}

	fmt.Printf("Найдено %d ошибок за последний час\n", len(entries))
}
*/
