package logger

// Константы для embedded систем - жестко заданные оптимальные значения
const (
	// Форматирование
	DEFAULT_TIME_FORMAT = "02-01-2006 15:04:05" // Фиксированный формат времени

	// Производительность
	DEFAULT_WRITE_BATCH_SIZE   = 50   // Оптимальный размер пакета для flash
	DEFAULT_MAX_CONNECTIONS    = 10   // Ограничение для embedded CPU (уменьшено с 20)
	DEFAULT_MAX_MESSAGE_SIZE   = 2048 // 2KB максимум на сообщение (уменьшено с 4KB)
	DEFAULT_CONNECTION_TIMEOUT = 30   // 30 секунд таймаут

	// Кеширование
	DEFAULT_CACHE_SIZE = 100    // 100 записей в кеше (уменьшено с 500)
	DEFAULT_CACHE_TTL  = 5 * 60 // 5 минут TTL

	// Безопасность
	DEFAULT_FILE_PERMISSIONS   = 0644 // Стандартные права для файлов
	DEFAULT_SOCKET_PERMISSIONS = 0666 // Стандартные права для сокетов
	DEFAULT_RATE_LIMIT         = 50   // 50 сообщений в секунду (уменьшено со 100)

	// Ресурсы
	DEFAULT_MAX_MEMORY = 50 * 1024 * 1024 // 50MB лимит памяти
)
