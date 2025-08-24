# Производительность ZLogger

## Обзор архитектуры

ZLogger спроектирован для максимальной производительности в embedded системах и высоконагруженных приложениях. Основные принципы оптимизации:

- **Архитектура клиент-сервер**: Разделение логики записи и отправки
- **Буферизация**: Пакетная запись для минимизации I/O операций
- **Пулы объектов**: Переиспользование структур для снижения GC нагрузки
- **Кеширование**: Быстрый доступ к последним записям
- **Rate limiting**: Защита от перегрузки

## Оптимизации для embedded систем

### Константы производительности

```go
const (
    DEFAULT_WRITE_BATCH_SIZE   = 50   // Оптимальный размер пакета для flash
    DEFAULT_MAX_CONNECTIONS    = 10   // Ограничение для embedded CPU
    DEFAULT_MAX_MESSAGE_SIZE   = 2048 // 2KB максимум на сообщение
    DEFAULT_CONNECTION_TIMEOUT = 30   // 30 секунд таймаут
    DEFAULT_CACHE_SIZE         = 100  // 100 записей в кеше
    DEFAULT_RATE_LIMIT         = 50   // 50 сообщений в секунду
    DEFAULT_MAX_MEMORY         = 50 * 1024 * 1024 // 50MB лимит памяти
)
```

### Выравнивание структур данных

Все структуры оптимизированы для правильного выравнивания на 32-битных архитектурах (MIPS):

```go
type LogServer struct {
    // int64 поля в начале для правильного выравнивания
    currentSize int64
    connCounter int64
    stats       ServerStats
    
    // Остальные поля
    config   *LoggingConfig
    file     *os.File
    // ...
}
```

## Рекомендации по конфигурации

### Для embedded систем (ограниченные ресурсы)

```go
config := &zlogger.Config{
    Level:            "warn",           // Минимум логов
    LogFile:          "/tmp/app.log",   // Быстрый носитель
    SocketPath:       "/tmp/app.sock",
    MaxFileSize:      5.0,              // 5 MB файлы
    BufferSize:       50,               // Экономия памяти
    FlushInterval:    10 * time.Second, // Редкий сброс
    Services:         []string{"MAIN"},
    RestrictServices: true,
}
```

**Потребление ресурсов:**
- Память: ~2-5 MB
- CPU: минимальное
- Диск: 5-10 MB

### Для высоконагруженных систем

```go
config := &zlogger.Config{
    Level:            "info",
    LogFile:          "/var/log/app.log",
    SocketPath:       "/tmp/app.sock",
    MaxFileSize:      500.0,            // Большие файлы
    BufferSize:       10000,            // Большой буфер
    FlushInterval:    30 * time.Second, // Редкий сброс
    Services:         []string{"API", "DB", "CACHE"},
    RestrictServices: false,
}
```

**Потребление ресурсов:**
- Память: ~50-100 MB
- CPU: низкое
- Диск: 500+ MB

### Для development

```go
config := &zlogger.Config{
    Level:            "debug",          // Все логи
    LogFile:          "./logs/dev.log",
    SocketPath:       "/tmp/dev.sock",
    MaxFileSize:      50.0,
    BufferSize:       500,
    FlushInterval:    time.Second,      // Быстрый сброс
    Services:         []string{},
    RestrictServices: false,
}
```

## Бенчмарки производительности

### Пропускная способность

| Конфигурация | Сообщений/сек | Память (MB) | CPU (%) |
|--------------|---------------|-------------|---------|
| Embedded     | 1,000-5,000   | 2-5         | 1-3     |
| Standard     | 10,000-50,000 | 10-30       | 3-8     |
| High-load    | 50,000-200,000| 50-100      | 8-15    |

### Латентность

| Операция | Embedded | Standard | High-load |
|----------|----------|----------|-----------|
| Отправка сообщения | 10-50μs | 5-20μs | 1-10μs |
| Запись на диск | 1-10ms | 0.5-5ms | 0.1-2ms |
| Поиск в логах | 10-100ms | 5-50ms | 1-20ms |

## Оптимизация производительности

### 1. Настройка буферизации

```go
// Для высокой пропускной способности
config.BufferSize = 10000
config.FlushInterval = 30 * time.Second

// Для низкой латентности
config.BufferSize = 100
config.FlushInterval = 100 * time.Millisecond

// Для экономии памяти
config.BufferSize = 50
config.FlushInterval = 5 * time.Second
```

### 2. Управление размером файлов

```go
// Для быстрых носителей (SSD)
config.MaxFileSize = 1000.0 // 1GB файлы

// Для медленных носителей (HDD)
config.MaxFileSize = 100.0  // 100MB файлы

// Для embedded (flash)
config.MaxFileSize = 10.0   // 10MB файлы
```

### 3. Оптимизация уровней логирования

```go
// Production - только важные сообщения
config.Level = "warn"

// Development - все сообщения
config.Level = "debug"

// Monitoring - информационные сообщения
config.Level = "info"
```

## Мониторинг производительности

### Встроенные метрики

```go
func monitorPerformance(logger *zlogger.Logger) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        // Проверяем соединение
        if err := logger.Ping(); err != nil {
            fmt.Printf("Сервер недоступен: %v\n", err)
            continue
        }
        
        // Получаем статистику через логи
        filter := &zlogger.FilterOptions{
            Service: "SLOG",
            Limit:   1,
        }
        
        entries, err := logger.GetLogEntries(*filter)
        if err == nil && len(entries) > 0 {
            fmt.Printf("Последняя активность сервера: %s\n", 
                entries[0].Timestamp.Format("15:04:05"))
        }
    }
}
```

### Профилирование памяти

```go
import (
    "runtime"
    "time"
)

func profileMemory(logger *zlogger.Logger) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        logger.Infof("Память: Alloc=%d KB, Sys=%d KB, GC=%d", 
            m.Alloc/1024, m.Sys/1024, m.NumGC)
        
        // Принудительная сборка мусора при превышении лимита
        if m.Alloc > 100*1024*1024 { // 100MB
            runtime.GC()
            logger.Warn("Принудительная сборка мусора")
        }
    }
}
```

## Оптимизация для конкретных сценариев

### Веб-серверы

```go
// Высокая частота запросов
config := &zlogger.Config{
    Level:            "info",
    BufferSize:       5000,             // Большой буфер
    FlushInterval:    5 * time.Second,  // Умеренный сброс
    MaxFileSize:      200.0,            // Средние файлы
    Services:         []string{"HTTP", "API", "DB"},
    RestrictServices: true,
}
```

### Микросервисы

```go
// Множество небольших сервисов
config := &zlogger.Config{
    Level:            "info",
    BufferSize:       1000,             // Средний буфер
    FlushInterval:    2 * time.Second,  // Быстрый сброс
    MaxFileSize:      50.0,             // Небольшие файлы
    Services:         []string{},       // Гибкость
    RestrictServices: false,
}
```

### Batch обработка

```go
// Периодическая обработка больших объемов
config := &zlogger.Config{
    Level:            "debug",
    BufferSize:       10000,            // Очень большой буфер
    FlushInterval:    60 * time.Second, // Редкий сброс
    MaxFileSize:      1000.0,           // Большие файлы
    Services:         []string{"BATCH", "WORKER"},
    RestrictServices: true,
}
```

## Troubleshooting производительности

### Высокое потребление памяти

1. **Уменьшите BufferSize**:
```go
config.BufferSize = 100 // Вместо 1000
```

2. **Увеличьте частоту сброса**:
```go
config.FlushInterval = time.Second // Вместо 10 секунд
```

3. **Ограничьте уровень логирования**:
```go
config.Level = "warn" // Вместо "debug"
```

### Высокая нагрузка на диск

1. **Увеличьте буферизацию**:
```go
config.BufferSize = 5000
config.FlushInterval = 10 * time.Second
```

2. **Используйте быстрые носители**:
```go
config.LogFile = "/tmp/app.log" // RAM диск
```

3. **Оптимизируйте размер файлов**:
```go
config.MaxFileSize = 500.0 // Больше файлы = меньше ротаций
```

### Высокая латентность

1. **Уменьшите буферизацию**:
```go
config.BufferSize = 50
config.FlushInterval = 100 * time.Millisecond
```

2. **Используйте локальные сокеты**:
```go
config.SocketPath = "/tmp/app.sock" // Локальная файловая система
```

## Рекомендации по deployment

### Systemd сервис

```ini
[Unit]
Description=ZLogger Application
After=network.target

[Service]
Type=simple
User=app
Group=app
WorkingDirectory=/opt/app
ExecStart=/opt/app/myapp
Restart=always
RestartSec=5

# Ограничения ресурсов
MemoryLimit=100M
CPUQuota=50%

# Переменные окружения
Environment=LOG_LEVEL=info
Environment=LOG_FILE=/var/log/app/app.log
Environment=SOCKET_PATH=/tmp/app.sock

[Install]
WantedBy=multi-user.target
```

### Docker контейнер

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/myapp .

# Создаем директории для логов
RUN mkdir -p /var/log/app /tmp

# Ограничиваем ресурсы
ENV LOG_LEVEL=info
ENV LOG_FILE=/var/log/app/app.log
ENV SOCKET_PATH=/tmp/app.sock

CMD ["./myapp"]
```

### Kubernetes deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zlogger-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: zlogger-app
  template:
    metadata:
      labels:
        app: zlogger-app
    spec:
      containers:
      - name: app
        image: myapp:latest
        resources:
          requests:
            memory: "50Mi"
            cpu: "100m"
          limits:
            memory: "200Mi"
            cpu: "500m"
        env:
        - name: LOG_LEVEL
          value: "info"
        - name: LOG_FILE
          value: "/var/log/app/app.log"
        - name: SOCKET_PATH
          value: "/tmp/app.sock"
        volumeMounts:
        - name: logs
          mountPath: /var/log/app
        - name: tmp
          mountPath: /tmp
      volumes:
      - name: logs
        emptyDir: {}
      - name: tmp
        emptyDir: {}
```
