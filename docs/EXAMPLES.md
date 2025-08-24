# Примеры использования ZLogger

## Базовые примеры

### Простое логирование

```go
package main

import (
    "github.com/qzeleza/zlogger"
)

func main() {
    // Создаем конфигурацию
    config := zlogger.NewConfig("/var/log/app.log", "/tmp/app.sock")
    
    // Создаем логгер
    logger, err := zlogger.New(config)
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    // Базовое логирование
    logger.Info("Приложение запущено")
    logger.Error("Произошла ошибка")
    logger.Debug("Отладочная информация")
}
```

### Логирование с форматированием

```go
func demonstrateFormatting(logger *zlogger.Logger) {
    userID := 12345
    duration := 150
    
    // Автоматическое форматирование
    logger.Info("Пользователь %d выполнил запрос за %d мс", userID, duration)
    
    // Явное форматирование
    logger.Infof("Запрос от пользователя %d занял %d мс", userID, duration)
    
    // Сложное форматирование
    logger.Debugf("Детали: метод=%s, путь=%s, IP=%s, код=%d", 
        "GET", "/api/users", "192.168.1.100", 200)
}
```

## Многосервисная архитектура

### Веб-приложение с микросервисами

```go
package main

import (
    "time"
    "github.com/qzeleza/zlogger"
)

func main() {
    // Конфигурация для микросервисной архитектуры
    config := &zlogger.Config{
        Level:            "info",
        LogFile:          "/var/log/webapp/app.log",
        SocketPath:       "/tmp/webapp.sock",
        MaxFileSize:      200.0,
        BufferSize:       2000,
        FlushInterval:    time.Second,
        Services:         []string{"API", "AUTH", "DATABASE", "CACHE", "WORKER"},
        RestrictServices: true,
    }
    
    logger, err := zlogger.New(config, "API", "AUTH", "DATABASE", "CACHE", "WORKER")
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    // Запускаем сервисы
    go apiService(logger)
    go authService(logger)
    go databaseService(logger)
    go cacheService(logger)
    go workerService(logger)
    
    // Основной цикл приложения
    logger.Info("Веб-приложение запущено")
    time.Sleep(10 * time.Second)
    logger.Info("Веб-приложение завершает работу")
}

func apiService(logger *zlogger.Logger) {
    apiLogger := logger.SetService("API")
    
    for i := 0; i < 5; i++ {
        apiLogger.Info("Получен HTTP запрос GET /users/%d", i+1)
        time.Sleep(500 * time.Millisecond)
        apiLogger.Info("HTTP ответ отправлен: 200 OK")
        time.Sleep(200 * time.Millisecond)
    }
}

func authService(logger *zlogger.Logger) {
    authLogger := logger.SetService("AUTH")
    
    authLogger.Info("Сервис аутентификации запущен")
    time.Sleep(1 * time.Second)
    authLogger.Warn("Неудачная попытка входа для пользователя: admin")
    time.Sleep(2 * time.Second)
    authLogger.Info("Успешная аутентификация пользователя: user123")
}

func databaseService(logger *zlogger.Logger) {
    dbLogger := logger.SetService("DATABASE")
    
    dbLogger.Info("Подключение к базе данных установлено")
    for i := 0; i < 3; i++ {
        dbLogger.Debug("Выполнение SQL запроса: SELECT * FROM users WHERE id = %d", i+1)
        time.Sleep(300 * time.Millisecond)
        dbLogger.Info("Запрос выполнен успешно, получено %d записей", (i+1)*10)
        time.Sleep(700 * time.Millisecond)
    }
}

func cacheService(logger *zlogger.Logger) {
    cacheLogger := logger.SetService("CACHE")
    
    cacheLogger.Info("Кеш-сервис инициализирован")
    time.Sleep(800 * time.Millisecond)
    cacheLogger.Info("Кеш обновлен для ключа: users_list")
    time.Sleep(1200 * time.Millisecond)
    cacheLogger.Infof("Статистика кеша: попаданий=%d, промахов=%d, коэффициент=%.2f", 
        850, 150, 0.85)
}

func workerService(logger *zlogger.Logger) {
    workerLogger := logger.SetService("WORKER")
    
    workerLogger.Info("Фоновый worker запущен")
    for i := 0; i < 2; i++ {
        time.Sleep(2 * time.Second)
        workerLogger.Info("Обработана задача #%d", i+1)
    }
}
```

## Обработка ошибок и исключений

### Graceful error handling

```go
func demonstrateErrorHandling(logger *zlogger.Logger) {
    dbLogger := logger.SetService("DATABASE")
    
    // Обработка ошибок подключения
    err := connectToDatabase()
    if err != nil {
        dbLogger.Errorf("Ошибка подключения к БД: %v", err)
        
        // Попытки переподключения
        for attempt := 1; attempt <= 3; attempt++ {
            dbLogger.Infof("Попытка переподключения %d/3", attempt)
            time.Sleep(time.Duration(attempt) * time.Second)
            
            if err := connectToDatabase(); err == nil {
                dbLogger.Info("Переподключение успешно")
                break
            } else {
                dbLogger.Warnf("Попытка %d неудачна: %v", attempt, err)
            }
        }
    }
}

// Обработка паники с логированием
func demonstratePanicHandling(logger *zlogger.Logger) {
    defer func() {
        if r := recover(); r != nil {
            logger.Errorf("Восстановлено после паники: %v", r)
            // Дополнительная логика восстановления
        }
    }()
    
    // Код, который может вызвать панику
    riskyOperation()
}

func connectToDatabase() error {
    // Имитация ошибки подключения
    return fmt.Errorf("connection timeout")
}

func riskyOperation() {
    // Имитация операции, которая может вызвать панику
    panic("критическая ошибка в операции")
}
```

## Мониторинг и метрики

### Система мониторинга

```go
package main

import (
    "time"
    "github.com/qzeleza/zlogger"
)

type MetricsCollector struct {
    logger *zlogger.Logger
    ticker *time.Ticker
    done   chan bool
}

func NewMetricsCollector(logger *zlogger.Logger) *MetricsCollector {
    return &MetricsCollector{
        logger: logger,
        ticker: time.NewTicker(30 * time.Second),
        done:   make(chan bool),
    }
}

func (m *MetricsCollector) Start() {
    metricsLogger := m.logger.SetService("METRICS")
    metricsLogger.Info("Система мониторинга запущена")
    
    go func() {
        for {
            select {
            case <-m.ticker.C:
                m.collectMetrics()
            case <-m.done:
                m.ticker.Stop()
                metricsLogger.Info("Система мониторинга остановлена")
                return
            }
        }
    }()
}

func (m *MetricsCollector) Stop() {
    m.done <- true
}

func (m *MetricsCollector) collectMetrics() {
    metricsLogger := m.logger.SetService("METRICS")
    
    // Имитация сбора метрик
    cpuUsage := 45.7
    memoryUsage := 67.2
    diskUsage := 23.1
    
    metricsLogger.Infof("CPU: %.1f%%, Memory: %.1f%%, Disk: %.1f%%", 
        cpuUsage, memoryUsage, diskUsage)
    
    // Проверка пороговых значений
    if cpuUsage > 80.0 {
        metricsLogger.Warn("Высокая загрузка CPU: %.1f%%", cpuUsage)
    }
    
    if memoryUsage > 90.0 {
        metricsLogger.Error("Критическое использование памяти: %.1f%%", memoryUsage)
    }
}

func main() {
    config := zlogger.NewConfig("/var/log/monitoring.log", "/tmp/monitoring.sock")
    logger, err := zlogger.New(config, "METRICS", "ALERTS")
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    collector := NewMetricsCollector(logger)
    collector.Start()
    
    // Работаем 2 минуты
    time.Sleep(2 * time.Minute)
    
    collector.Stop()
    time.Sleep(100 * time.Millisecond) // Ждем остановки
}
```

## Фильтрация и поиск логов

### Анализ логов

```go
func demonstrateLogAnalysis(logger *zlogger.Logger) {
    // Ждем накопления логов
    time.Sleep(1 * time.Second)
    
    // Поиск ошибок за последний час
    oneHourAgo := time.Now().Add(-1 * time.Hour)
    errorLevel := zlogger.ERROR
    
    errorFilter := &zlogger.FilterOptions{
        StartTime: &oneHourAgo,
        Level:     &errorLevel,
        Limit:     100,
    }
    
    errors, err := logger.GetLogEntries(*errorFilter)
    if err != nil {
        logger.Errorf("Ошибка получения логов: %v", err)
        return
    }
    
    logger.Infof("Найдено %d ошибок за последний час", len(errors))
    
    // Анализ по сервисам
    serviceStats := make(map[string]int)
    for _, entry := range errors {
        serviceStats[entry.Service]++
    }
    
    for service, count := range serviceStats {
        logger.Infof("Сервис %s: %d ошибок", service, count)
    }
    
    // Поиск записей конкретного сервиса
    apiFilter := &zlogger.FilterOptions{
        Service: "API",
        Limit:   50,
    }
    
    apiLogs, err := logger.GetLogEntries(*apiFilter)
    if err != nil {
        logger.Errorf("Ошибка получения логов API: %v", err)
        return
    }
    
    logger.Infof("Последние %d записей API сервиса:", len(apiLogs))
    for i, entry := range apiLogs {
        if i < 5 { // Показываем только первые 5
            logger.Infof("  [%s] %s: %s", 
                entry.Level.String(), 
                entry.Timestamp.Format("15:04:05"), 
                entry.Message)
        }
    }
}
```

## Интеграция с HTTP сервером

### Middleware для логирования HTTP запросов

```go
package main

import (
    "fmt"
    "net/http"
    "time"
    "github.com/qzeleza/zlogger"
)

type LoggingMiddleware struct {
    logger *zlogger.ServiceLogger
}

func NewLoggingMiddleware(logger *zlogger.Logger) *LoggingMiddleware {
    return &LoggingMiddleware{
        logger: logger.SetService("HTTP"),
    }
}

func (lm *LoggingMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Логируем входящий запрос
        lm.logger.Infof("Входящий запрос: %s %s от %s", 
            r.Method, r.URL.Path, r.RemoteAddr)
        
        // Создаем wrapper для ResponseWriter чтобы захватить статус код
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        // Выполняем запрос
        next.ServeHTTP(wrapped, r)
        
        // Логируем результат
        duration := time.Since(start)
        lm.logger.Infof("Запрос завершен: %s %s - %d (%v)", 
            r.Method, r.URL.Path, wrapped.statusCode, duration)
        
        // Логируем медленные запросы
        if duration > 1*time.Second {
            lm.logger.Warnf("Медленный запрос: %s %s - %v", 
                r.Method, r.URL.Path, duration)
        }
        
        // Логируем ошибки
        if wrapped.statusCode >= 400 {
            lm.logger.Errorf("Ошибка HTTP: %s %s - %d", 
                r.Method, r.URL.Path, wrapped.statusCode)
        }
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func main() {
    config := zlogger.NewConfig("/var/log/webserver.log", "/tmp/webserver.sock")
    logger, err := zlogger.New(config, "HTTP", "API")
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    // Создаем middleware
    loggingMiddleware := NewLoggingMiddleware(logger)
    
    // Настраиваем роуты
    mux := http.NewServeMux()
    
    mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
        apiLogger := logger.SetService("API")
        apiLogger.Info("Обработка запроса пользователей")
        
        // Имитация обработки
        time.Sleep(100 * time.Millisecond)
        
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"users": [{"id": 1, "name": "John"}]}`)
    })
    
    mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
        apiLogger := logger.SetService("API")
        apiLogger.Error("Тестовая ошибка API")
        
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    })
    
    // Применяем middleware
    handler := loggingMiddleware.Handler(mux)
    
    logger.Info("HTTP сервер запущен на порту 8080")
    http.ListenAndServe(":8080", handler)
}
```

## Конфигурация для разных сред

### Development конфигурация

```go
func developmentConfig() *zlogger.Config {
    return &zlogger.Config{
        Level:            "debug",
        LogFile:          "./logs/dev.log",
        SocketPath:       "/tmp/dev.sock",
        MaxFileSize:      10.0,
        BufferSize:       100,
        FlushInterval:    500 * time.Millisecond,
        Services:         []string{},
        RestrictServices: false,
    }
}
```

### Production конфигурация

```go
func productionConfig() *zlogger.Config {
    return &zlogger.Config{
        Level:            "info",
        LogFile:          "/var/log/app/production.log",
        SocketPath:       "/tmp/production.sock",
        MaxFileSize:      500.0,
        BufferSize:       5000,
        FlushInterval:    5 * time.Second,
        Services:         []string{"API", "DB", "CACHE", "AUTH"},
        RestrictServices: true,
    }
}
```

### Embedded системы

```go
func embeddedConfig() *zlogger.Config {
    return &zlogger.Config{
        Level:            "warn",
        LogFile:          "/tmp/embedded.log",
        SocketPath:       "/tmp/embedded.sock",
        MaxFileSize:      5.0,
        BufferSize:       50,
        FlushInterval:    10 * time.Second,
        Services:         []string{"MAIN", "SENSOR"},
        RestrictServices: true,
    }
}
```

## Тестирование с ZLogger

### Unit тесты

```go
package main

import (
    "testing"
    "time"
    "github.com/qzeleza/zlogger"
)

func TestLogging(t *testing.T) {
    // Создаем временную конфигурацию для тестов
    config := &zlogger.Config{
        Level:         "debug",
        LogFile:       "/tmp/test.log",
        SocketPath:    "/tmp/test.sock",
        MaxFileSize:   1.0,
        BufferSize:    10,
        FlushInterval: 100 * time.Millisecond,
    }
    
    logger, err := zlogger.New(config)
    if err != nil {
        t.Fatalf("Ошибка создания логгера: %v", err)
    }
    defer logger.Close()
    
    // Тестируем базовое логирование
    logger.Info("Тестовое сообщение")
    logger.Error("Тестовая ошибка")
    
    // Ждем записи
    time.Sleep(200 * time.Millisecond)
    
    // Проверяем, что сообщения записались
    filter := &zlogger.FilterOptions{Limit: 10}
    entries, err := logger.GetLogEntries(*filter)
    if err != nil {
        t.Fatalf("Ошибка получения записей: %v", err)
    }
    
    if len(entries) < 2 {
        t.Errorf("Ожидалось минимум 2 записи, получено %d", len(entries))
    }
    
    // Проверяем содержимое
    found := false
    for _, entry := range entries {
        if entry.Message == "Тестовое сообщение" {
            found = true
            break
        }
    }
    
    if !found {
        t.Error("Тестовое сообщение не найдено в логах")
    }
}
```
