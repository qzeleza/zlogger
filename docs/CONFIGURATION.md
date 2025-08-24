# Конфигурация ZLogger

## Обзор

ZLogger использует структуру `Config` для настройки всех параметров системы логирования. Конфигурация позволяет оптимизировать производительность под конкретные требования приложения.

## Структура конфигурации

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

## Параметры конфигурации

### Level (string)

Минимальный уровень логирования. Сообщения ниже указанного уровня будут игнорироваться.

**Возможные значения:**
- `"debug"` - все сообщения
- `"info"` - информационные сообщения и выше
- `"warn"` - предупреждения и ошибки
- `"error"` - только ошибки и критические сообщения
- `"fatal"` - только критические ошибки
- `"panic"` - только паника

**Пример:**
```go
config.Level = "info"
```

### LogFile (string)

Полный путь к файлу лога. Директория будет создана автоматически, если не существует.

**Рекомендации:**
- Используйте абсолютные пути
- Убедитесь, что у процесса есть права на запись
- Для embedded систем выбирайте быстрые носители

**Пример:**
```go
config.LogFile = "/var/log/myapp/app.log"
```

### SocketPath (string)

Путь к Unix сокету для межпроцессного взаимодействия.

**Рекомендации:**
- Используйте директорию `/tmp` для временных сокетов
- Убедитесь, что путь уникален для приложения
- Директория должна быть доступна для записи

**Пример:**
```go
config.SocketPath = "/tmp/myapp.sock"
```

### MaxFileSize (float64)

Максимальный размер лог файла в мегабайтах. При достижении лимита происходит ротация.

**Рекомендации:**
- Для embedded систем: 10-50 MB
- Для серверных приложений: 100-500 MB
- Учитывайте доступное дисковое пространство

**Пример:**
```go
config.MaxFileSize = 100.0 // 100 MB
```

### BufferSize (int)

Размер буфера сообщений в памяти. Больший буфер улучшает производительность, но увеличивает потребление памяти.

**Рекомендации:**
- Для embedded систем: 100-1000
- Для высоконагруженных систем: 1000-10000
- Баланс между производительностью и памятью

**Пример:**
```go
config.BufferSize = 1000
```

### FlushInterval (time.Duration)

Интервал принудительного сброса буфера на диск. Меньший интервал обеспечивает лучшую надежность, но снижает производительность.

**Рекомендации:**
- Для критичных приложений: 100ms-1s
- Для обычных приложений: 1s-5s
- Для высокопроизводительных систем: 5s-30s

**Пример:**
```go
config.FlushInterval = 2 * time.Second
```

### Services ([]string)

Список разрешенных сервисов для логирования. Используется совместно с `RestrictServices`.

**Пример:**
```go
config.Services = []string{"API", "DATABASE", "CACHE", "AUTH"}
```

### RestrictServices (bool)

Ограничивает логирование только указанными в `Services` сервисами.

**Рекомендации:**
- `true` - для production среды с контролем сервисов
- `false` - для development и гибкой настройки

**Пример:**
```go
config.RestrictServices = true
```

## Создание конфигурации

### Базовая конфигурация

```go
config := zlogger.NewConfig("/var/log/app.log", "/tmp/app.sock")
```

### Полная настройка

```go
config := &zlogger.Config{
    Level:            "info",
    LogFile:          "/var/log/myapp/app.log",
    SocketPath:       "/tmp/myapp.sock",
    MaxFileSize:      100.0,
    BufferSize:       1000,
    FlushInterval:    time.Second,
    Services:         []string{"API", "DB", "CACHE"},
    RestrictServices: true,
}
```

## Рекомендации по настройке

### Для embedded систем

```go
config := &zlogger.Config{
    Level:            "warn",           // Минимум логов
    LogFile:          "/tmp/app.log",   // Быстрый носитель
    SocketPath:       "/tmp/app.sock",
    MaxFileSize:      10.0,             // Небольшие файлы
    BufferSize:       100,              // Экономия памяти
    FlushInterval:    5 * time.Second,  // Редкий сброс
    Services:         []string{"MAIN"},
    RestrictServices: true,
}
```

### Для высоконагруженных систем

```go
config := &zlogger.Config{
    Level:            "info",
    LogFile:          "/var/log/app/app.log",
    SocketPath:       "/tmp/app.sock",
    MaxFileSize:      500.0,            // Большие файлы
    BufferSize:       10000,            // Большой буфер
    FlushInterval:    10 * time.Second, // Редкий сброс
    Services:         []string{"API", "DB", "CACHE", "AUTH", "WORKER"},
    RestrictServices: false,            // Гибкость
}
```

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
    RestrictServices: false,            // Без ограничений
}
```

## Валидация конфигурации

ZLogger автоматически валидирует конфигурацию при создании логгера:

- `LogFile` не может быть пустым
- `SocketPath` не может быть пустым  
- `Level` должен быть валидным уровнем
- `MaxFileSize` должен быть положительным
- `BufferSize` должен быть положительным

## Изменение конфигурации во время работы

```go
// Создаем новую конфигурацию
newConfig := &zlogger.Config{
    Level:         "debug",
    LogFile:       "/new/path/app.log",
    SocketPath:    "/tmp/new.sock",
    MaxFileSize:   200.0,
    BufferSize:    2000,
    FlushInterval: 500 * time.Millisecond,
}

// Обновляем конфигурацию
err := logger.UpdateConfig(newConfig)
if err != nil {
    log.Printf("Ошибка обновления конфигурации: %v", err)
}
```

## Переменные окружения

Можно использовать переменные окружения для настройки:

```go
func configFromEnv() *zlogger.Config {
    config := zlogger.NewConfig(
        getEnv("LOG_FILE", "/var/log/app.log"),
        getEnv("SOCKET_PATH", "/tmp/app.sock"),
    )
    
    config.Level = getEnv("LOG_LEVEL", "info")
    
    if size := getEnv("MAX_FILE_SIZE", ""); size != "" {
        if f, err := strconv.ParseFloat(size, 64); err == nil {
            config.MaxFileSize = f
        }
    }
    
    return config
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

**Пример использования:**
```bash
export LOG_LEVEL=debug
export LOG_FILE=/var/log/myapp.log
export MAX_FILE_SIZE=200
./myapp
```
