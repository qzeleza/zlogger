# Тестирование модуля Logger

Этот документ описывает комплексную систему тестирования модуля logger, оптимизированную для embedded устройств.

## Структура тестов

### Unit тесты
- `logger_test.go` - основные функции логгера
- `levels_test.go` - тестирование уровней логирования
- `message_test.go` - структуры сообщений и memory pooling
- `service_test.go` - тестирование ServiceLogger
- `mocks_test.go` - мок объекты для тестирования

### Integration тесты
- `integration_test.go` - интеграционные тесты с мок сервером

### Performance тесты
- `performance_test.go` - тесты производительности и памяти
- `embedded_test.go` - специальные тесты для embedded устройств

## Запуск тестов

### Все тесты
```bash
make test
```

### Unit тесты только
```bash
make test-unit
```

### Интеграционные тесты
```bash
make test-integration
```

### Тесты производительности
```bash
make test-performance
```

### Embedded тесты
```bash
make test-embedded
```

### Покрытие кода
```bash
make test-coverage
```

## Команды Go

### Базовые команды

```bash
# Все тесты
go test ./...

# Только unit тесты (быстрые)
go test -short ./...

# Verbose режим
go test -v ./...

# Конкретный тест
go test -run TestLoggerMethods ./...

# Бенчмарки
go test -bench=. ./...

# Покрытие
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Embedded тесты

```bash
# Тесты для embedded устройств
go test -tags embedded ./...

# С ограничением памяти
GOMAXPROCS=1 go test -tags embedded ./...

# С профилированием памяти
go test -tags embedded -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### Тесты производительности

```bash
# Все бенчмарки
go test -bench=. -benchmem ./...

# Конкретный бенчмарк
go test -bench=BenchmarkLogMessagePool -benchmem ./...

# Длительные бенчмарки
go test -bench=. -benchtime=10s ./...

# CPU профилирование
go test -bench=. -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

## Целевые показатели

### Покрытие кода
- **Unit тесты**: > 85%
- **Integration тесты**: > 70%
- **Общее покрытие**: > 80%

### Производительность
- **Пропускная способность**: > 100 ops/sec на embedded устройствах
- **Использование памяти**: < 10MB heap для embedded тестов
- **Время запуска**: < 100ms для embedded устройств

### Ресурсы embedded систем
- **Максимальная память**: 50MB (DEFAULT_MAX_MEMORY)
- **Максимальные соединения**: 20 (DEFAULT_MAX_CONNECTIONS)
- **Размер сообщения**: 4KB (DEFAULT_MAX_MESSAGE_SIZE)
- **Rate limit**: 100 msg/sec (DEFAULT_RATE_LIMIT)

## Типы тестов

### 1. Unit тесты
Тестируют отдельные компоненты в изоляции:
- Создание логгера
- Методы логирования всех уровней
- Валидация параметров
- Memory pooling
- Кеширование ServiceLogger'ов

### 2. Integration тесты
Тестируют взаимодействие компонентов:
- Клиент-серверное взаимодействие
- Переподключение при разрыве соединения
- Параллельное логирование
- Получение записей лога

### 3. Performance тесты
Проверяют производительность и использование ресурсов:
- Пропускная способность
- Использование памяти
- Утечки памяти
- Параллельная производительность

### 4. Memory тесты
Специальные тесты для проверки памяти:
- Проверка утечек памяти
- Эффективность memory pooling
- Работа в условиях ограниченной памяти
- GC pressure тесты

### 5. Embedded тесты
Тесты для embedded устройств:
- Ограниченные ресурсы процессора
- Жесткие лимиты памяти
- Энергоэффективность
- Минимизация записи в flash память

## Мок объекты

### MockLogClient
Полнофункциональный мок клиента логгера:
- Отслеживание вызовов методов
- Имитация ошибок сети
- Кеширование ServiceLogger'ов
- Потокобезопасность

### MockServer
Простой мок сервер для интеграционных тестов:
- Unix socket сервер
- Обработка протокольных сообщений
- Хранение полученных сообщений
- Graceful shutdown

## Отладка тестов

### Verbose режим
```bash
go test -v ./... | grep -E "(PASS|FAIL|RUN)"
```

### Конкретный тест с отладкой
```bash
go test -v -run TestLoggerMethods ./...
```

### Профилирование памяти
```bash
go test -memprofile=mem.prof -run TestMemoryUsage ./...
go tool pprof mem.prof
```

### Профилирование CPU
```bash
go test -cpuprofile=cpu.prof -bench=BenchmarkHighThroughput ./...
go tool pprof cpu.prof
```

## CI/CD интеграция

### GitHub Actions пример
```yaml
- name: Run tests
  run: |
    go test -short ./...
    go test -race ./...
    go test -bench=. -benchtime=1s ./...
    go test -tags embedded ./...
```

### Проверки качества
```bash
# Линтинг
golangci-lint run ./...

# Форматирование
gofmt -s -w .

# Модули
go mod tidy
go mod verify
```

## Troubleshooting

### Проблемы с сокетами
```bash
# Очистка временных сокетов
rm -f /tmp/test_*.sock
```

### Проблемы с памятью
```bash
# Увеличение лимита памяти для тестов
GOMAXPROCS=2 go test ./...
```

### Медленные тесты
```bash
# Пропуск интеграционных тестов
go test -short ./...

# Таймаут для тестов
go test -timeout=30s ./...
```

## Метрики качества

### Покрытие по файлам
- `logger.go`: > 90%
- `client.go`: > 80% (сложность сетевого кода)
- `service.go`: > 95%
- `message.go`: > 90%
- `levels.go`: > 95%

### Производительность
- Memory pooling: < 100ns per operation
- Level comparison: < 10ns per operation
- Service caching: < 50ns per operation

### Надежность
- Zero memory leaks в длительных тестах
- Graceful degradation под нагрузкой
- Корректная обработка ошибок сети
