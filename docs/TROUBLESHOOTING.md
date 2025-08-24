# Устранение неполадок ZLogger

## Частые проблемы и решения

### Проблемы с подключением

#### Ошибка: "connection refused"

**Симптомы:**
```
Ошибка подключения к серверу логгера: dial unix /tmp/app.sock: connect: connection refused
```

**Причины и решения:**

1. **Сервер логгера не запущен**
```go
// Проверьте, что сервер создан и запущен
logger, err := zlogger.New(config)
if err != nil {
    log.Printf("Ошибка создания логгера: %v", err)
}
```

2. **Неверный путь к сокету**
```go
// Убедитесь, что путь существует и доступен для записи
config.SocketPath = "/tmp/myapp.sock"
```

3. **Права доступа к сокету**
```bash
# Проверьте права доступа
ls -la /tmp/myapp.sock
# Должно быть: srw-rw-rw-
```

#### Ошибка: "no such file or directory"

**Решение:**
```go
// Создайте директорию для сокета
socketDir := filepath.Dir(config.SocketPath)
err := os.MkdirAll(socketDir, 0755)
if err != nil {
    return fmt.Errorf("ошибка создания директории: %w", err)
}
```

### Проблемы с файлами логов

#### Ошибка: "permission denied"

**Симптомы:**
```
Ошибка инициализации файла лога: open /var/log/app.log: permission denied
```

**Решения:**

1. **Создайте директорию с правильными правами**
```bash
sudo mkdir -p /var/log/myapp
sudo chown $USER:$USER /var/log/myapp
sudo chmod 755 /var/log/myapp
```

2. **Используйте доступную директорию**
```go
config.LogFile = "/tmp/app.log" // Вместо /var/log/app.log
```

3. **Запустите с правами sudo** (не рекомендуется)
```bash
sudo ./myapp
```

#### Ошибка: "no space left on device"

**Решения:**

1. **Уменьшите размер файлов**
```go
config.MaxFileSize = 10.0 // 10 MB вместо 100 MB
```

2. **Очистите старые логи**
```bash
find /var/log -name "*.log" -mtime +7 -delete
```

3. **Используйте другую директорию**
```go
config.LogFile = "/tmp/app.log"
```

### Проблемы с производительностью

#### Высокое потребление памяти

**Диагностика:**
```go
func checkMemory() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    fmt.Printf("Память: %d MB\n", m.Alloc/1024/1024)
}
```

**Решения:**

1. **Уменьшите размер буфера**
```go
config.BufferSize = 100 // Вместо 1000
```

2. **Увеличьте частоту сброса**
```go
config.FlushInterval = time.Second // Вместо 10 секунд
```

3. **Ограничьте уровень логирования**
```go
config.Level = "warn" // Вместо "debug"
```

#### Медленная запись логов

**Диагностика:**
```go
start := time.Now()
logger.Info("Тестовое сообщение")
duration := time.Since(start)
fmt.Printf("Время записи: %v\n", duration)
```

**Решения:**

1. **Увеличьте буферизацию**
```go
config.BufferSize = 5000
config.FlushInterval = 10 * time.Second
```

2. **Используйте быстрый носитель**
```go
config.LogFile = "/tmp/app.log" // RAM диск
```

3. **Оптимизируйте размер сообщений**
```go
// Избегайте очень длинных сообщений
logger.Info("Краткое сообщение")
```

### Проблемы с конфигурацией

#### Ошибка: "invalid log level"

**Симптомы:**
```
невалидный уровень логирования 'INVALID': неизвестный уровень логирования: INVALID
```

**Решение:**
```go
// Используйте правильные уровни
validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
config.Level = "info"
```

#### Ошибка: "buffer size too small"

**Решение:**
```go
// Минимальный размер буфера
if config.BufferSize < 10 {
    config.BufferSize = 10
}
```

### Проблемы с многопоточностью

#### Race conditions

**Диагностика:**
```bash
go run -race myapp.go
```

**Решение:**
```go
// ZLogger thread-safe по умолчанию
// Но избегайте одновременного изменения конфигурации
var mu sync.Mutex

func updateConfig(logger *zlogger.Logger, newConfig *zlogger.Config) error {
    mu.Lock()
    defer mu.Unlock()
    return logger.UpdateConfig(newConfig)
}
```

#### Deadlocks

**Избегайте:**
```go
// НЕ ДЕЛАЙТЕ ТАК - может вызвать deadlock
func badExample(logger *zlogger.Logger) {
    defer func() {
        if r := recover(); r != nil {
            logger.Error("Паника: %v", r) // Может заблокироваться
        }
    }()
    
    logger.Info("Начало операции")
    panic("тест")
}
```

**Правильно:**
```go
func goodExample(logger *zlogger.Logger) {
    defer func() {
        if r := recover(); r != nil {
            // Используйте отдельный канал или goroutine
            go func() {
                logger.Error("Паника: %v", r)
            }()
        }
    }()
    
    logger.Info("Начало операции")
}
```

## Отладка

### Включение отладочного режима

```go
config := &zlogger.Config{
    Level: "debug", // Включаем все сообщения
    // ... остальные параметры
}
```

### Проверка состояния сервера

```go
func healthCheck(logger *zlogger.Logger) error {
    // Проверяем соединение
    if err := logger.Ping(); err != nil {
        return fmt.Errorf("сервер недоступен: %w", err)
    }
    
    // Проверяем запись
    testMsg := fmt.Sprintf("Тест %d", time.Now().Unix())
    if err := logger.Info(testMsg); err != nil {
        return fmt.Errorf("ошибка записи: %w", err)
    }
    
    // Проверяем чтение
    time.Sleep(100 * time.Millisecond)
    filter := &zlogger.FilterOptions{Limit: 1}
    entries, err := logger.GetLogEntries(*filter)
    if err != nil {
        return fmt.Errorf("ошибка чтения: %w", err)
    }
    
    if len(entries) == 0 {
        return fmt.Errorf("записи не найдены")
    }
    
    return nil
}
```

### Мониторинг ресурсов

```go
func monitorResources(logger *zlogger.Logger) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        // Логируем статистику
        logger.Infof("Память: Alloc=%d KB, Sys=%d KB, NumGC=%d", 
            m.Alloc/1024, m.Sys/1024, m.NumGC)
        
        // Проверяем лимиты
        if m.Alloc > 100*1024*1024 { // 100MB
            logger.Warn("Высокое потребление памяти")
            runtime.GC()
        }
        
        // Проверяем количество горутин
        numGoroutines := runtime.NumGoroutine()
        if numGoroutines > 100 {
            logger.Warnf("Много горутин: %d", numGoroutines)
        }
    }
}
```

## Инструменты диагностики

### Анализ логов

```bash
# Поиск ошибок
grep "ERROR" /var/log/app.log

# Статистика по уровням
awk -F'[\\[\\]]' '{print $4}' /var/log/app.log | sort | uniq -c

# Активность по времени
awk '{print $3}' /var/log/app.log | cut -d: -f1-2 | sort | uniq -c
```

### Мониторинг сокетов

```bash
# Проверка активных сокетов
lsof | grep "\.sock"

# Статистика сетевых соединений
ss -x | grep app.sock
```

### Мониторинг производительности

```bash
# CPU и память процесса
top -p $(pgrep myapp)

# I/O статистика
iotop -p $(pgrep myapp)

# Открытые файлы
lsof -p $(pgrep myapp)
```

## Решение специфических проблем

### Embedded системы

**Проблема: Недостаток памяти**
```go
config := &zlogger.Config{
    Level:         "error",    // Только ошибки
    BufferSize:    10,         // Минимальный буфер
    MaxFileSize:   1.0,        // 1MB файлы
    FlushInterval: time.Second, // Быстрый сброс
}
```

**Проблема: Медленный flash**
```go
config := &zlogger.Config{
    BufferSize:    100,               // Больше буферизации
    FlushInterval: 30 * time.Second,  // Редкие записи
    MaxFileSize:   5.0,               // Небольшие файлы
}
```

### Контейнеры

**Проблема: Потеря логов при рестарте**
```yaml
# Docker Compose
volumes:
  - ./logs:/var/log/app
  - /tmp:/tmp
```

**Проблема: Права доступа**
```dockerfile
RUN adduser -D -s /bin/sh appuser
USER appuser
```

### Kubernetes

**Проблема: Ephemeral storage**
```yaml
spec:
  containers:
  - name: app
    volumeMounts:
    - name: logs
      mountPath: /var/log/app
  volumes:
  - name: logs
    persistentVolumeClaim:
      claimName: app-logs
```

## Получение поддержки

### Сбор диагностической информации

```go
func collectDiagnostics(logger *zlogger.Logger) {
    fmt.Println("=== ZLogger Diagnostics ===")
    
    // Версия Go
    fmt.Printf("Go version: %s\n", runtime.Version())
    
    // Архитектура
    fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
    
    // Память
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    fmt.Printf("Memory: Alloc=%d KB, Sys=%d KB\n", m.Alloc/1024, m.Sys/1024)
    
    // Горутины
    fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
    
    // Проверка соединения
    if err := logger.Ping(); err != nil {
        fmt.Printf("Connection: FAILED (%v)\n", err)
    } else {
        fmt.Printf("Connection: OK\n")
    }
    
    // Последние логи
    filter := &zlogger.FilterOptions{Limit: 5}
    entries, err := logger.GetLogEntries(*filter)
    if err != nil {
        fmt.Printf("Log retrieval: FAILED (%v)\n", err)
    } else {
        fmt.Printf("Log retrieval: OK (%d entries)\n", len(entries))
    }
}
```

### Создание минимального воспроизводимого примера

```go
package main

import (
    "fmt"
    "time"
    "github.com/qzeleza/zlogger"
)

func main() {
    // Минимальная конфигурация
    config := zlogger.NewConfig("/tmp/test.log", "/tmp/test.sock")
    
    logger, err := zlogger.New(config)
    if err != nil {
        fmt.Printf("Ошибка: %v\n", err)
        return
    }
    defer logger.Close()
    
    // Воспроизведение проблемы
    logger.Info("Тестовое сообщение")
    
    // Ожидание и проверка
    time.Sleep(100 * time.Millisecond)
    
    filter := &zlogger.FilterOptions{Limit: 1}
    entries, err := logger.GetLogEntries(*filter)
    if err != nil {
        fmt.Printf("Ошибка получения логов: %v\n", err)
        return
    }
    
    fmt.Printf("Получено %d записей\n", len(entries))
}
```
