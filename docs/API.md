# API Reference

## Основные типы

### Logger

Основной интерфейс логгера для клиентских приложений.

```go
type Logger struct {
    // внутренние поля
}
```

### ServiceLogger

Логгер для конкретного сервиса с предустановленным именем.

```go
type ServiceLogger struct {
    // внутренние поля
}
```

### LogLevel

Уровни логирования с числовыми значениями для быстрого сравнения.

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

### Config

Конфигурация системы логирования.

```go
type Config struct {
    Level            string        // Уровень логирования
    LogFile          string        // Путь к лог файлу
    SocketPath       string        // Путь к Unix сокету
    MaxFileSize      float64       // Максимальный размер файла в MB
    BufferSize       int           // Размер буфера сообщений
    FlushInterval    time.Duration // Интервал сброса буфера
    Services         []string      // Список разрешенных сервисов
    RestrictServices bool          // Ограничить сервисы
}
```

## Основные функции

### New

Создает новый экземпляр логгера.

```go
func New(config *Config, services ...string) (*Logger, error)
```

**Параметры:**
- `config` - конфигурация логгера (обязательный)
- `services` - список дополнительных сервисов (опционально)

**Возвращает:**
- `*Logger` - экземпляр логгера
- `error` - ошибка инициализации

**Пример:**
```go
config := zlogger.NewConfig("/var/log/app.log", "/tmp/app.sock")
logger, err := zlogger.New(config, "API", "DATABASE", "CACHE")
if err != nil {
    return err
}
defer logger.Close()
```

### NewConfig

Создает конфигурацию с настройками по умолчанию.

```go
func NewConfig(logFile, socketPath string) *Config
```

**Параметры:**
- `logFile` - путь к файлу лога
- `socketPath` - путь к Unix сокету

**Возвращает:**
- `*Config` - готовая конфигурация

### ParseLevel

Парсит строковый уровень логирования.

```go
func ParseLevel(level string) (LogLevel, error)
```

**Параметры:**
- `level` - строковое представление уровня

**Возвращает:**
- `LogLevel` - уровень логирования
- `error` - ошибка парсинга

## Методы Logger

### Основные методы логирования

```go
func (l *Logger) Debug(message string, args ...interface{}) error
func (l *Logger) Info(message string, args ...interface{}) error
func (l *Logger) Warn(message string, args ...interface{}) error
func (l *Logger) Error(message string, args ...interface{}) error
func (l *Logger) Fatal(message string, args ...interface{}) error
func (l *Logger) Panic(message string, args ...interface{}) error
```

**Параметры:**
- `message` - текст сообщения
- `args` - аргументы для форматирования (опционально)

**Пример:**
```go
logger.Info("Пользователь вошел в систему")
logger.Error("Ошибка подключения к БД: %v", err)
```

### Форматированные методы

```go
func (l *Logger) Debugf(format string, args ...interface{}) error
func (l *Logger) Infof(format string, args ...interface{}) error
func (l *Logger) Warnf(format string, args ...interface{}) error
func (l *Logger) Errorf(format string, args ...interface{}) error
func (l *Logger) Fatalf(format string, args ...interface{}) error
func (l *Logger) Panicf(format string, args ...interface{}) error
```

**Параметры:**
- `format` - строка форматирования
- `args` - аргументы для форматирования

**Пример:**
```go
logger.Infof("Пользователь %d выполнил запрос за %d мс", userID, duration)
```

### Управление сервисами

#### SetService

Возвращает логгер для указанного сервиса.

```go
func (l *Logger) SetService(service string) *ServiceLogger
```

**Параметры:**
- `service` - имя сервиса

**Возвращает:**
- `*ServiceLogger` - логгер для сервиса

**Пример:**
```go
apiLogger := logger.SetService("API")
apiLogger.Info("HTTP запрос обработан")
```

### Управление уровнями

#### SetLevel

Устанавливает локальный уровень логирования.

```go
func (l *Logger) SetLevel(level LogLevel)
```

#### SetServerLevel

Устанавливает уровень логирования на сервере.

```go
func (l *Logger) SetServerLevel(level LogLevel) error
```

### Получение записей

#### GetLogEntries

Получает записи из лога с фильтрацией.

```go
func (l *Logger) GetLogEntries(filter FilterOptions) ([]LogEntry, error)
```

**Параметры:**
- `filter` - опции фильтрации

**Возвращает:**
- `[]LogEntry` - массив записей лога
- `error` - ошибка получения

### Служебные методы

#### Ping

Проверяет соединение с сервером.

```go
func (l *Logger) Ping() error
```

#### Close

Закрывает логгер и освобождает ресурсы.

```go
func (l *Logger) Close() error
```

## Методы ServiceLogger

ServiceLogger имеет те же методы логирования, что и Logger, но все сообщения автоматически помечаются именем сервиса.

```go
func (s *ServiceLogger) Debug(message string) error
func (s *ServiceLogger) Info(message string) error
func (s *ServiceLogger) Warn(message string) error
func (s *ServiceLogger) Error(message string) error
func (s *ServiceLogger) Fatal(message string) error
func (s *ServiceLogger) Panic(message string) error

func (s *ServiceLogger) Debugf(format string, args ...interface{}) error
func (s *ServiceLogger) Infof(format string, args ...interface{}) error
func (s *ServiceLogger) Warnf(format string, args ...interface{}) error
func (s *ServiceLogger) Errorf(format string, args ...interface{}) error
func (s *ServiceLogger) Fatalf(format string, args ...interface{}) error
func (s *ServiceLogger) Panicf(format string, args ...interface{}) error
```

## Глобальные функции

Для быстрого логирования без создания экземпляра (выводят в stdout/stderr):

```go
func Debug(message string)
func Info(message string)
func Warn(message string)
func Error(message string)
func Fatal(message string)

func Debugf(format string, args ...interface{})
func Infof(format string, args ...interface{})
func Warnf(format string, args ...interface{})
func Errorf(format string, args ...interface{})
func Fatalf(format string, args ...interface{})
```

## Типы для фильтрации

### LogEntry

Запись лога для чтения.

```go
type LogEntry struct {
    Service   string    // Название сервиса
    Level     LogLevel  // Уровень логирования
    Message   string    // Текст сообщения
    Timestamp time.Time // Время создания
    Raw       string    // Исходная строка лога
}
```

### FilterOptions

Опции фильтрации логов.

```go
type FilterOptions struct {
    StartTime *time.Time // Начальное время фильтрации
    EndTime   *time.Time // Конечное время фильтрации
    Level     *LogLevel  // Фильтр по уровню
    Service   string     // Фильтр по сервису
    Limit     int        // Лимит количества записей
}
```

**Пример использования:**
```go
filter := &zlogger.FilterOptions{
    Service: "API",
    Level:   &[]zlogger.LogLevel{zlogger.ERROR}[0],
    Limit:   100,
}

entries, err := logger.GetLogEntries(*filter)
if err != nil {
    return err
}

for _, entry := range entries {
    fmt.Printf("[%s] %s: %s\n", entry.Service, entry.Level, entry.Message)
}
```
