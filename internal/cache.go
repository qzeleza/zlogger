// cache.go - Система кеширования для повышения производительности
package logger

import (
	"container/list"
	"runtime"
	"sync"
	"time"
)

// LogCache кеш записей лога в памяти для быстрого доступа
type LogCache struct {
	mu      sync.RWMutex             // Мьютекс для безопасного доступа
	entries *list.List               // Связанный список записей (LRU)
	lookup  map[string]*list.Element // Быстрый поиск по ключу
	maxSize int                      // Максимальный размер кеша
	ttl     time.Duration            // Время жизни записей
	stats   CacheStats               // Статистика кеша
	done    chan struct{}            // Канал для остановки cleanup горутины
}

// CacheEntry элемент кеша с метаданными
type CacheEntry struct {
	Key       string    // Ключ записи
	Entry     LogEntry  // Запись лога
	Timestamp time.Time // Время добавления в кеш
}

// CacheStats статистика работы кеша
type CacheStats struct {
	Hits      int64 // Количество попаданий
	Misses    int64 // Количество промахов
	Evictions int64 // Количество вытеснений
	Size      int   // Текущий размер кеша
}

// NewLogCache создает новый кеш записей лога
func NewLogCache(maxSize int, ttl time.Duration) *LogCache {
	cache := &LogCache{
		entries: list.New(),
		lookup:  make(map[string]*list.Element),
		maxSize: maxSize,
		ttl:     ttl,
		done:    make(chan struct{}),
	}

	// Запускаем фоновую очистку устаревших записей
	if ttl > 0 {
		go cache.cleanupExpired()
	}

	// Регистрируем финалайзер, чтобы гарантировать остановку goroutine,
	// даже если пользователь кэша забудет вызвать Close().
	runtime.SetFinalizer(cache, func(c *LogCache) {
		c.Close()
	})

	return cache
}

// Get получает запись из кеша
func (c *LogCache) Get(key string) (*LogEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, found := c.lookup[key]
	if !found {
		c.stats.Misses++
		return nil, false
	}

	entry := element.Value.(*CacheEntry)

	// Проверяем TTL
	if c.ttl > 0 && time.Since(entry.Timestamp) > c.ttl {
		c.removeElement(element)
		c.stats.Misses++
		return nil, false
	}

	// Перемещаем элемент в начало (LRU)
	c.entries.MoveToFront(element)
	c.stats.Hits++

	return &entry.Entry, true
}

// Put добавляет запись в кеш
func (c *LogCache) Put(key string, entry LogEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Проверяем, существует ли уже такой ключ
	if element, found := c.lookup[key]; found {
		// Обновляем существующую запись
		cacheEntry := element.Value.(*CacheEntry)
		cacheEntry.Entry = entry
		cacheEntry.Timestamp = time.Now()
		c.entries.MoveToFront(element)
		return
	}

	// Создаем новую запись
	cacheEntry := &CacheEntry{
		Key:       key,
		Entry:     entry,
		Timestamp: time.Now(),
	}

	element := c.entries.PushFront(cacheEntry)
	c.lookup[key] = element
	c.stats.Size++

	// Проверяем лимит размера кеша
	if c.stats.Size > c.maxSize {
		c.evictOldest()
	}
}

// evictOldest удаляет самую старую запись из кеша
func (c *LogCache) evictOldest() {
	element := c.entries.Back()
	if element != nil {
		c.removeElement(element)
		c.stats.Evictions++
	}
}

// removeElement удаляет элемент из кеша
func (c *LogCache) removeElement(element *list.Element) {
	entry := element.Value.(*CacheEntry)
	delete(c.lookup, entry.Key)
	c.entries.Remove(element)
	c.stats.Size--
}

// cleanupExpired очищает устаревшие записи в фоне
func (c *LogCache) cleanupExpired() {
	ticker := time.NewTicker(c.ttl / 2) // Проверяем дважды за TTL
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return // Завершаем горутину
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()

			// Проходим по списку с конца (самые старые)
			for element := c.entries.Back(); element != nil; {
				entry := element.Value.(*CacheEntry)
				if now.Sub(entry.Timestamp) > c.ttl {
					prev := element.Prev()
					c.removeElement(element)
					c.stats.Evictions++
					element = prev
				} else {
					break // Остальные записи еще актуальны
				}
			}

			c.mu.Unlock()
		}
	}
}

// GetStats возвращает статистику кеша
func (c *LogCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// Clear очищает весь кеш
func (c *LogCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries.Init()
	c.lookup = make(map[string]*list.Element)
	c.stats.Size = 0
}

// Close останавливает cleanup горутину LogCache
func (c *LogCache) Close() {
    // Защита от двойного закрытия канала
    select {
    case <-c.done:
        return
    default:
        close(c.done)
    }
}
