# Logger Module - Developer Documentation

## Обзор архитектуры

Модуль logger реализует высокопроизводительную систему логирования для embedded устройств с архитектурой клиент-сервер. Система оптимизирована для минимального потребления ресурсов и максимальной производительности на устройствах с ограниченными возможностями.

### Ключевые особенности

- **Архитектура клиент-сервер** - разделение на клиентскую и серверную части через Unix сокеты
- **Оптимизация для embedded** - минимальное потребление памяти и CPU
- **Буферизованная запись** - пакетная обработка сообщений для повышения производительности
- **Кеширование** - встроенный LRU кеш для быстрого доступа к записям
- **Безопасность** - защита от DoS атак через rate limiting
- **Мультисервисная поддержка** - возможность логирования от разных сервисов

### Форматы данных

Система использует два формата данных для разных целей:

1. **JSON для IPC** (межпроцессное взаимодействие):
   ```json
   {"service":"DNS","level":1,"message":"Запрос обработан","timestamp":"..."}
   ```

2. **TXT для файла лога** (человекочитаемый формат):
   ```
   [DNS  ] 02-01-2024 14:30:23 [INFO ] "Запрос обработан"
   ```

## Архитектурные компоненты

### 1. Основные интерфейсы

#### API Interface (`api.go`)
Упрощенный интерфейс для внешнего использования:

```go
type API interface {
    SetService(service string) *ServiceLogger
    Debugf(string, ...interface{}) error
    Errorf(string, ...interface{}) error
    Infof(string, ...interface{}) error
    Warnf(string, ...interface{}) error
    Debug(string) error
    Info(string) error
    Warn(string) error
    Error(string) error
}
```

#### LogClientInterface (`interfaces.go`)
Полный интерфейс клиента с расширенными возможностями:

```go
type LogClientInterface interface {
    // Управление сервисами
    SetService(service string) *ServiceLogger
    SetLevel(level LogLevel)
    SetServerLevel(level LogLevel) error
    
    // Управление конфигурацией
    GetLogFile() string
    UpdateConfig(config *conf.LoggingConfig) error
    
    // Получение данных
    GetLogEntries(filter FilterOptions) ([]LogEntry, error)
    
    // Служебные методы
    Ping() error
    Close() error
    LogPanic()
    
    // Методы логирования
    Debug/Info/Warn/Error/Fatal/Panic(message string) error
    Debugf/Infof/Warnf/Errorf/Fatalf/Panicf(format string, args ...interface{}) error
}
```

### 2. Основные структуры

#### Logger (`logger.go`)
Главная обертка для клиентских приложений:

```go
type Logger struct {
    client LogClientInterface
}

// Создание с автозапуском сервера
func New(config *conf.LoggingConfig, services []string) (*Logger, error)
```

#### LogClient (`client.go`)
Клиентская часть с полным функционалом:

```go
type LogClient struct {
    config         *conf.LoggingConfig
    conn           net.Conn
    encoder        *json.Encoder
    decoder        *json.Decoder
    mu             sync.Mutex
    level          LogLevel
    reconnectMu    sync.Mutex
    serviceLoggers map[string]*ServiceLogger  // Кеш логгеров сервисов
    servicesMu     sync.RWMutex
    connected      bool
}
```

#### ServiceLogger (`service.go`)
Логгер для конкретного сервиса:

```go
type ServiceLogger struct {
    client  LogClientInterface
    service string
}
```

#### LogServer (`server.go`)
Серверная часть с оптимизациями:

```go
type LogServer struct {
    // Основная конфигурация
    config   *config.LoggingConfig
    file     *os.File
    listener net.Listener
    
    // Буферизация и производительность
    buffer     chan LogMessage
    writeBatch []LogMessage
    batchMu    sync.Mutex
    
    // Управление жизненным циклом
    done    chan struct{}
    stopped bool
    wg      sync.WaitGroup
    
    // Метрики и мониторинг
    currentSize   int64
    maxServiceLen int
    maxLevelLen   int
    
    // Управление клиентами
    clients     map[net.Conn]string
    clientsMu   sync.RWMutex
    connCounter int64
    
    // Фильтрация и безопасность
    minLevel       LogLevel
    rateLimiter    *RateLimiter
    securityConfig *SecurityConfig
    
    // Кеширование
    cache *LogCache
    
    // Статистика
    stats ServerStats
}
```

### 3. Система уровней логирования

#### LogLevel (`levels.go`)
Оптимизированная система уровней:

```go
type LogLevel int

const (
    DEBUG LogLevel = iota // 0 - Отладочная информация
    INFO                  // 1 - Информационные сообщения
    WARN                  // 2 - Предупреждения
    ERROR                 // 3 - Ошибки
    FATAL                 // 4 - Критические ошибки
    PANIC                 // 5 - Паника приложения
)
```

**Особенности реализации:**
- Числовые значения для быстрого сравнения
- Кешированные строковые представления
- Быстрый парсинг через map lookup

### 4. Система сообщений

#### Структуры сообщений (`message.go`)

```go
// Для IPC между клиентом и сервером
type LogMessage struct {
    Service   string    `json:"service"`
    Level     LogLevel  `json:"level"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
    ClientID  string    `json:"client_id,omitempty"`
}

// Для чтения из файла лога
type LogEntry struct {
    Service   string    `json:"service"`
    Level     LogLevel  `json:"level"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
    Raw       string    `json:"raw"`  // Исходная строка
}

// Протокольное сообщение для IPC
type ProtocolMessage struct {
    Type string      `json:"type"`
    Data interface{} `json:"data"`
}
```

**Оптимизации:**
- Пулы объектов для переиспользования (`sync.Pool`)
- Минимизация аллокаций через `GetLogMessage()` / `PutLogMessage()`

### 5. Система кеширования

#### LogCache (`cache.go`)
LRU кеш с TTL для быстрого доступа:

```go
type LogCache struct {
    mu      sync.RWMutex
    entries *list.List               // LRU список
    lookup  map[string]*list.Element // Быстрый поиск
    maxSize int                      // Максимальный размер
    ttl     time.Duration            // Время жизни
    stats   CacheStats               // Статистика
    done    chan struct{}            // Остановка cleanup
}
```

**Функциональность:**
- LRU вытеснение при переполнении
- TTL для автоматической очистки устаревших записей
- Фоновая очистка expired записей
- Статистика hit/miss ratio

### 6. Система безопасности

#### SecurityConfig (`security.go`)
Защита от атак и некорректного использования:

```go
type SecurityConfig struct {
    MaxMessageLength    int            // Максимальная длина сообщения
    MaxServiceLength    int            // Максимальная длина имени сервиса
    AllowedServiceChars *regexp.Regexp // Разрешенные символы
    RateLimitPerSecond  int            // Ограничение скорости
    BanDuration         time.Duration  // Длительность бана
}

type RateLimiter struct {
    clients map[string]*ClientInfo // Информация о клиентах
    mu      sync.RWMutex          // Потокобезопасность
    config  *SecurityConfig       // Конфигурация
    done    chan struct{}         // Остановка cleanup
}
```

**Защитные механизмы:**
- Rate limiting по клиентам
- Валидация длины и содержимого сообщений
- Временные баны за превышение лимитов
- Защита от path traversal атак

## Жизненный цикл системы

### 1. Инициализация

```go
// Создание логгера с автозапуском сервера
config := &conf.LoggingConfig{
    LogFile:    "/var/log/app.log",
    SocketPath: "/tmp/logger.sock",
    Level:      "info",
    BufferSize: 1000,
    // ... другие параметры
}

logger, err := logger.New(config, []string{"DNS", "HTTP", "API"})
if err != nil {
    return err
}
defer logger.Close()
```

### 2. Использование основного логгера

```go
// Логирование от MAIN сервиса
logger.Info("Приложение запущено")
logger.Errorf("Ошибка подключения: %v", err)

// Получение логгера для конкретного сервиса
dnsLogger := logger.SetService("DNS")
dnsLogger.Debug("Обработка DNS запроса")
dnsLogger.Warn("Превышен таймаут DNS")
```

### 3. Серверная часть

```go
// Ручное создание сервера (обычно не требуется)
server, err := logger.NewLogServer(config)
if err != nil {
    return err
}

// Запуск сервера
if err := server.Start(); err != nil {
    return err
}
defer server.Stop()
```

### 4. Работа с записями лога

```go
// Получение записей с фильтрацией
filter := logger.FilterOptions{
    Level:   &logger.INFO,
    Service: "DNS",
    Limit:   1000,
}

entries, err := logger.GetLogEntries(filter)
if err != nil {
    return err
}

for _, entry := range entries {
    fmt.Printf("[%s] %s: %s\n", entry.Service, entry.Level, entry.Message)
}
```

## Оптимизации для embedded систем

### 1. Константы производительности (`defaults.go`)

```go
const (
    // Буферизация
    DEFAULT_WRITE_BATCH_SIZE   = 50    // Оптимальный размер пакета для flash
    DEFAULT_MAX_CONNECTIONS    = 10    // Ограничение для embedded CPU
    DEFAULT_MAX_MESSAGE_SIZE   = 2048  // 2KB максимум на сообщение
    
    // Кеширование
    DEFAULT_CACHE_SIZE = 100     // 100 записей в кеше
    DEFAULT_CACHE_TTL  = 5 * 60  // 5 минут TTL
    
    // Безопасность
    DEFAULT_RATE_LIMIT = 50      // 50 сообщений в секунду
    
    // Ресурсы
    DEFAULT_MAX_MEMORY = 50 * 1024 * 1024  // 50MB лимит памяти
)
```

### 2. Буферизованная запись

Система использует пакетную запись в файл для минимизации системных вызовов:

```go
// Буфер сообщений
buffer := make(chan LogMessage, config.BufferSize)

// Пакетная обработка
writeBatch := make([]LogMessage, 0, DEFAULT_WRITE_BATCH_SIZE)

// Периодическая запись пакетами
for {
    select {
    case msg := <-buffer:
        writeBatch = append(writeBatch, msg)
        if len(writeBatch) >= DEFAULT_WRITE_BATCH_SIZE {
            server.flushBatch(writeBatch)
            writeBatch = writeBatch[:0]  // Переиспользуем слайс
        }
    case <-flushTimer.C:
        if len(writeBatch) > 0 {
            server.flushBatch(writeBatch)
            writeBatch = writeBatch[:0]
        }
    }
}
```

### 3. Переиспользование объектов

```go
// Пулы объектов для минимизации GC pressure
var logMessagePool = sync.Pool{
    New: func() interface{} {
        return &LogMessage{}
    },
}

// Использование
msg := GetLogMessage()
defer PutLogMessage(msg)
```

### 4. Эффективное переподключение

```go
// Экспоненциальный backoff для переподключения
backoff := time.Millisecond * 100
maxBackoff := time.Second * 10
maxAttempts := 5

for attempt := 0; attempt < maxAttempts; attempt++ {
    if err := client.connect(); err == nil {
        return nil
    }
    
    time.Sleep(backoff)
    backoff *= 2
    if backoff > maxBackoff {
        backoff = maxBackoff
    }
}
```

## Протокол взаимодействия

### Типы сообщений

```go
const (
    MsgTypeLog         = "log"          // Сообщение лога
    MsgTypeGetEntries  = "get_entries"  // Запрос записей
    MsgTypeSetLevel    = "set_level"    // Установка уровня
    MsgTypeGetLogFile  = "get_log_file" // Получение пути к файлу
    MsgTypePing        = "ping"         // Проверка соединения
    MsgTypePong        = "pong"         // Ответ на ping
    MsgTypeError       = "error"        // Ошибка
    MsgTypeResponse    = "response"     // Ответ сервера
)
```

### Примеры протокольных сообщений

**Отправка лог-сообщения:**
```json
{
    "type": "log",
    "data": {
        "service": "DNS",
        "level": 1,
        "message": "Запрос обработан успешно",
        "timestamp": "2024-01-15T14:30:23Z"
    }
}
```

**Запрос записей:**
```json
{
    "type": "get_entries",
    "data": {
        "service": "DNS",
        "level": 1,
        "limit": 100,
        "start_time": "2024-01-15T14:00:00Z"
    }
}
```

**Ответ сервера:**
```json
{
    "type": "response",
    "data": [
        {
            "service": "DNS",
            "level": 1,
            "message": "Запрос обработан",
            "timestamp": "2024-01-15T14:30:23Z",
            "raw": "[DNS  ] 15-01-2024 14:30:23 [INFO ] \"Запрос обработан\""
        }
    ]
}
```

## Обработка ошибок и отказоустойчивость

### 1. Fallback механизмы

```go
// При недоступности сервера - запись в stderr
func (c *LogClient) fallbackToStderr(service string, level LogLevel, message string, timestamp time.Time) {
    serviceFormatted := fmt.Sprintf("%-5s", service)
    levelFormatted := fmt.Sprintf("%-5s", level.String())
    timeStr := timestamp.Format(DEFAULT_TIME_FORMAT)
    
    fmt.Fprintf(os.Stderr, "[%s] %s [%s] \"%s\"\n",
        serviceFormatted, timeStr, levelFormatted, message)
}
```

### 2. Автоматическое переподключение

```go
// Переподключение при обрыве соединения
func (c *LogClient) sendMessage(service string, level LogLevel, message string) error {
    // Проверяем соединение
    if !c.connected || c.conn == nil {
        if err := c.reconnect(); err != nil {
            c.fallbackToStderr(service, level, message, time.Now())
            return err
        }
    }
    
    // Отправляем сообщение
    if err := c.encoder.Encode(protocolMsg); err != nil {
        // Пытаемся переподключиться и повторить
        if reconnectErr := c.reconnect(); reconnectErr == nil {
            if retryErr := c.encoder.Encode(protocolMsg); retryErr == nil {
                return nil
            }
        }
        
        // Fallback при неудаче
        c.fallbackToStderr(service, level, message, time.Now())
        return err
    }
    
    return nil
}
```

### 3. Graceful shutdown

```go
// Корректное завершение работы сервера
func (s *LogServer) Stop() error {
    s.mu.Lock()
    if s.stopped {
        s.mu.Unlock()
        return nil
    }
    s.stopped = true
    s.mu.Unlock()
    
    // Сигнализируем о завершении
    close(s.done)
    
    // Ждем завершения горутин
    s.wg.Wait()
    
    // Записываем оставшиеся сообщения
    s.flushRemaining()
    
    // Закрываем ресурсы
    if s.listener != nil {
        s.listener.Close()
    }
    if s.file != nil {
        s.file.Close()
    }
    
    return nil
}
```

## Мониторинг и метрики

### 1. Статистика сервера

```go
type ServerStats struct {
    TotalMessages  int64     // Общее количество сообщений
    TotalClients   int64     // Общее количество подключений
    CurrentClients int32     // Текущее количество клиентов
    MemoryUsage    int64     // Использование памяти
    FileRotations  int64     // Количество ротаций файла
    CacheHits      int64     // Попадания в кеш
    CacheMisses    int64     // Промахи кеша
    LastRotation   time.Time // Последняя ротация
    StartTime      time.Time // Время запуска
}
```

### 2. Статистика кеша

```go
type CacheStats struct {
    Hits      int64 // Попадания
    Misses    int64 // Промахи
    Evictions int64 // Вытеснения
    Size      int   // Текущий размер
}
```

### 3. Мониторинг производительности

```go
// Получение статистики
stats := server.GetStats()
fmt.Printf("Сообщений обработано: %d\n", stats.TotalMessages)
fmt.Printf("Память: %d bytes\n", stats.MemoryUsage)
fmt.Printf("Cache hit ratio: %.2f%%\n", 
    float64(stats.CacheHits) / float64(stats.CacheHits + stats.CacheMisses) * 100)
```

## Лучшие практики разработки

### 1. Создание нового сервиса

```go
// Правильно - с указанием сервиса в конфигурации
config := &conf.LoggingConfig{
    Services: []string{"DNS", "HTTP", "API", "NEW_SERVICE"},
    // ... другие параметры
}

logger, err := logger.New(config, nil)
if err != nil {
    return err
}

// Получение логгера для нового сервиса
newServiceLogger := logger.SetService("NEW_SERVICE")
newServiceLogger.Info("Новый сервис запущен")
```

### 2. Обработка паник

```go
func riskyOperation() {
    defer logger.LogPanic()  // Автоматическое логирование паник
    
    // Рискованный код
    result := someOperation()
    if result == nil {
        panic("Critical error occurred")
    }
}
```

### 3. Производительность

```go
// Плохо - создание строки каждый раз
logger.Info("Processing item " + fmt.Sprintf("%d", itemID))

// Хорошо - использование форматирования
logger.Infof("Processing item %d", itemID)

// Еще лучше - проверка уровня логирования
if logger.GetLevel() <= DEBUG {
    expensiveDebugInfo := generateDebugInfo()  // Вызывается только если нужно
    logger.Debugf("Debug info: %s", expensiveDebugInfo)
}
```

### 4. Конфигурирование для production

```go
// Оптимальная конфигурация для production на embedded устройстве
config := &conf.LoggingConfig{
    LogFile:           "/var/log/kvaspro.log",
    SocketPath:        "/tmp/kvaspro_logger.sock",
    Level:             "info",  // Не debug в production
    BufferSize:        1000,    // Достаточный буфер
    MaxFileSize:       10,      // 10MB максимум
    MaxFiles:          3,       // 3 ротированных файла
    FlushInterval:     5000,    // 5 секунд flush
    Compress:          true,    // Сжатие старых файлов
    Services:          []string{"MAIN", "DNS", "HTTP", "API"},
}
```

### 5. Отладка и диагностика

```go
// Проверка соединения
if err := logger.Ping(); err != nil {
    log.Printf("Logger server unavailable: %v", err)
}

// Получение пути к файлу лога
logFile := logger.GetLogFile()
fmt.Printf("Logs are written to: %s\n", logFile)

// Получение записей для анализа
entries, err := logger.GetLogEntries(logger.FilterOptions{
    Level: &logger.ERROR,
    Limit: 100,
})
if err == nil {
    fmt.Printf("Found %d error entries\n", len(entries))
}
```

## Troubleshooting

### Частые проблемы и решения

1. **"connection refused" при создании логгера**
   - Проверьте права доступа к сокету
   - Убедитесь что директория для сокета существует
   - Проверьте что сервер логгера запущен

2. **Высокое потребление памяти**
   - Уменьшите `BufferSize` в конфигурации
   - Уменьшите `DEFAULT_CACHE_SIZE`
   - Проверьте на утечки логгеров сервисов

3. **Медленная запись в лог**
   - Увеличьте `DEFAULT_WRITE_BATCH_SIZE`
   - Уменьшите `FlushInterval`
   - Проверьте скорость записи на диск

4. **Переполнение буфера сообщений**
   - Увеличьте `BufferSize`
   - Оптимизируйте частоту логирования
   - Проверьте производительность диска

5. **Проблемы с ротацией файлов**
   - Проверьте права доступа к директории лога
   - Убедитесь в наличии свободного места
   - Проверьте настройки `MaxFiles` и `MaxFileSize`

### Диагностические команды

```bash
# Проверка сокета
ls -la /tmp/kvaspro_logger.sock

# Мониторинг размера лога
du -h /var/log/kvaspro.log*

# Проверка производительности записи
iostat -x 1

# Мониторинг памяти процесса
ps aux | grep kvaspro
```

## Тестирование

Модуль включает обширный набор тестов с покрытием более 90%:

- **Unit тесты** - тестирование отдельных компонентов
- **Integration тесты** - тестирование взаимодействия компонентов  
- **Performance тесты** - бенчмарки производительности
- **Security тесты** - тестирование защитных механизмов
- **Embedded тесты** - тестирование на ограниченных ресурсах

```bash
# Запуск всех тестов
make test

# Тесты с покрытием
make test-coverage

# Бенчмарки
go test -bench=. -benchmem ./internal/logger/
```

Система логирования готова к использованию в production среде embedded устройств с минимальными требованиями к ресурсам и максимальной производительностью.