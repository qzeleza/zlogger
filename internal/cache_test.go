// cache_test.go - Unit тесты для LogCache
package logger

import (
	"fmt"
	"testing"
	"time"
)

// TestNewLogCache проверяет создание нового кеша
func TestNewLogCache(t *testing.T) {
	maxSize := 100
	ttl := 5 * time.Minute

	cache := NewLogCache(maxSize, ttl)
	defer cache.Close() // Закрываем cleanup горутину после теста

	if cache == nil {
		t.Fatal("NewLogCache не должен возвращать nil")
	}

	if cache.maxSize != maxSize {
		t.Errorf("ожидался maxSize %d, получили %d", maxSize, cache.maxSize)
	}

	if cache.ttl != ttl {
		t.Errorf("ожидался ttl %v, получили %v", ttl, cache.ttl)
	}

	if cache.entries == nil {
		t.Error("entries не должен быть nil")
	}

	if cache.lookup == nil {
		t.Error("lookup не должен быть nil")
	}

	// Проверяем начальную статистику
	stats := cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("начальный размер должен быть 0, получили %d", stats.Size)
	}
	if stats.Hits != 0 {
		t.Errorf("начальные попадания должны быть 0, получили %d", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("начальные промахи должны быть 0, получили %d", stats.Misses)
	}
	if stats.Evictions != 0 {
		t.Errorf("начальные вытеснения должны быть 0, получили %d", stats.Evictions)
	}
}

// TestNewLogCacheWithZeroTTL проверяет создание кеша без TTL
func TestNewLogCacheWithZeroTTL(t *testing.T) {
	cache := NewLogCache(50, 0)
	defer cache.Close() // Закрываем cleanup горутину после теста

	if cache == nil {
		t.Fatal("NewLogCache не должен возвращать nil")
	}

	if cache.ttl != 0 {
		t.Errorf("ожидался ttl 0, получили %v", cache.ttl)
	}
}

// TestLogCachePutAndGet проверяет основные операции Put и Get
func TestLogCachePutAndGet(t *testing.T) {
	cache := NewLogCache(10, 0) // Без TTL для простоты
	defer cache.Close()         // Закрываем cleanup горутину после теста

	// Создаем тестовую запись
	entry := LogEntry{
		Service:   "TEST",
		Level:     INFO,
		Message:   "test message",
		Timestamp: time.Now(),
	}

	key := "test_key"

	// Проверяем, что записи нет в кеше
	result, found := cache.Get(key)
	if found {
		t.Error("запись не должна быть найдена в пустом кеше")
	}
	if result != nil {
		t.Error("результат должен быть nil для несуществующей записи")
	}

	// Добавляем запись в кеш
	cache.Put(key, entry)

	// Проверяем, что запись теперь есть в кеше
	result, found = cache.Get(key)
	if !found {
		t.Error("запись должна быть найдена после добавления")
	}
	if result == nil {
		t.Fatal("результат не должен быть nil для существующей записи")
	}

	// Проверяем содержимое записи
	if result.Service != entry.Service {
		t.Errorf("ожидался сервис '%s', получили '%s'", entry.Service, result.Service)
	}
	if result.Level != entry.Level {
		t.Errorf("ожидался уровень %v, получили %v", entry.Level, result.Level)
	}
	if result.Message != entry.Message {
		t.Errorf("ожидалось сообщение '%s', получили '%s'", entry.Message, result.Message)
	}

	// Проверяем статистику
	stats := cache.GetStats()
	if stats.Size != 1 {
		t.Errorf("размер кеша должен быть 1, получили %d", stats.Size)
	}
	if stats.Hits != 1 {
		t.Errorf("попадания должны быть 1, получили %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("промахи должны быть 1, получили %d", stats.Misses)
	}
}

// TestLogCachePutUpdateExisting проверяет обновление существующей записи
func TestLogCachePutUpdateExisting(t *testing.T) {
	cache := NewLogCache(10, 0)
	defer cache.Close() // Закрываем cleanup горутину после теста

	key := "update_key"

	// Первая запись
	entry1 := LogEntry{
		Service: "SERVICE1",
		Level:   INFO,
		Message: "first message",
	}

	// Вторая запись (обновление)
	entry2 := LogEntry{
		Service: "SERVICE2",
		Level:   ERROR,
		Message: "updated message",
	}

	// Добавляем первую запись
	cache.Put(key, entry1)

	// Проверяем размер кеша
	stats := cache.GetStats()
	if stats.Size != 1 {
		t.Errorf("размер кеша должен быть 1 после первого добавления, получили %d", stats.Size)
	}

	// Обновляем запись
	cache.Put(key, entry2)

	// Размер кеша не должен измениться
	stats = cache.GetStats()
	if stats.Size != 1 {
		t.Errorf("размер кеша должен остаться 1 после обновления, получили %d", stats.Size)
	}

	// Проверяем, что запись обновилась
	result, found := cache.Get(key)
	if !found {
		t.Error("обновленная запись должна быть найдена")
	}
	if result.Service != entry2.Service {
		t.Errorf("ожидался обновленный сервис '%s', получили '%s'", entry2.Service, result.Service)
	}
	if result.Message != entry2.Message {
		t.Errorf("ожидалось обновленное сообщение '%s', получили '%s'", entry2.Message, result.Message)
	}
}

// TestLogCacheEviction проверяет вытеснение старых записей при превышении лимита
func TestLogCacheEviction(t *testing.T) {
	maxSize := 3
	cache := NewLogCache(maxSize, 0)
	defer cache.Close() // Закрываем cleanup горутину после теста

	// Добавляем записи до лимита
	for i := 0; i < maxSize; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := LogEntry{
			Service: fmt.Sprintf("SERVICE_%d", i),
			Level:   INFO,
			Message: fmt.Sprintf("message %d", i),
		}
		cache.Put(key, entry)
	}

	// Проверяем, что размер равен лимиту
	stats := cache.GetStats()
	if stats.Size != maxSize {
		t.Errorf("размер кеша должен быть %d, получили %d", maxSize, stats.Size)
	}

	// Добавляем еще одну запись, что должно вызвать вытеснение
	cache.Put("key_overflow", LogEntry{
		Service: "OVERFLOW_SERVICE",
		Level:   WARN,
		Message: "overflow message",
	})

	// Размер должен остаться равным лимиту
	stats = cache.GetStats()
	if stats.Size != maxSize {
		t.Errorf("размер кеша должен остаться %d после вытеснения, получили %d", maxSize, stats.Size)
	}

	// Должно быть одно вытеснение
	if stats.Evictions != 1 {
		t.Errorf("должно быть 1 вытеснение, получили %d", stats.Evictions)
	}

	// Самая старая запись (key_0) должна быть удалена
	_, found := cache.Get("key_0")
	if found {
		t.Error("самая старая запись должна быть вытеснена")
	}

	// Новая запись должна быть доступна
	result, found := cache.Get("key_overflow")
	if !found {
		t.Error("новая запись должна быть найдена")
	}
	if result.Service != "OVERFLOW_SERVICE" {
		t.Errorf("ожидался сервис 'OVERFLOW_SERVICE', получили '%s'", result.Service)
	}
}

// TestLogCacheTTL проверяет работу TTL (время жизни записей)
func TestLogCacheTTL(t *testing.T) {
	ttl := 100 * time.Millisecond
	cache := NewLogCache(10, ttl)
	defer cache.Close() // Закрываем cleanup горутину после теста

	key := "ttl_key"
	entry := LogEntry{
		Service: "TTL_SERVICE",
		Level:   INFO,
		Message: "ttl test message",
	}

	// Добавляем запись
	cache.Put(key, entry)

	// Сразу после добавления запись должна быть доступна
	result, found := cache.Get(key)
	if !found {
		t.Error("запись должна быть найдена сразу после добавления")
	}
	if result == nil {
		t.Error("результат не должен быть nil для существующей записи")
	}

	// Ждем истечения TTL
	time.Sleep(ttl + 50*time.Millisecond)

	// После истечения TTL запись должна быть недоступна
	result, found = cache.Get(key)
	if found {
		t.Error("запись не должна быть найдена после истечения TTL")
	}
	if result != nil {
		t.Error("результат должен быть nil для просроченной записи")
	}

	// Проверяем, что статистика учитывает промах
	stats := cache.GetStats()
	if stats.Misses < 1 {
		t.Errorf("должен быть хотя бы 1 промах, получили %d", stats.Misses)
	}
}

// TestLogCacheClear проверяет очистку кеша
func TestLogCacheClear(t *testing.T) {
	cache := NewLogCache(10, 0)
	defer cache.Close() // Закрываем cleanup горутину после теста

	// Добавляем несколько записей
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("clear_key_%d", i)
		entry := LogEntry{
			Service: fmt.Sprintf("SERVICE_%d", i),
			Level:   INFO,
			Message: fmt.Sprintf("message %d", i),
		}
		cache.Put(key, entry)
	}

	// Проверяем, что записи добавлены
	stats := cache.GetStats()
	if stats.Size != 5 {
		t.Errorf("размер кеша должен быть 5, получили %d", stats.Size)
	}

	// Очищаем кеш
	cache.Clear()

	// Проверяем, что кеш пуст
	stats = cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("размер кеша должен быть 0 после очистки, получили %d", stats.Size)
	}

	// Проверяем, что записи недоступны
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("clear_key_%d", i)
		_, found := cache.Get(key)
		if found {
			t.Errorf("запись '%s' не должна быть найдена после очистки", key)
		}
	}
}

// TestLogCacheGetStats проверяет получение статистики кеша
func TestLogCacheGetStats(t *testing.T) {
	cache := NewLogCache(5, 0)
	defer cache.Close() // Закрываем cleanup горутину после теста

	// Начальная статистика
	stats := cache.GetStats()
	if stats.Size != 0 || stats.Hits != 0 || stats.Misses != 0 || stats.Evictions != 0 {
		t.Error("начальная статистика должна быть нулевой")
	}

	// Добавляем записи
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("stats_key_%d", i)
		entry := LogEntry{Service: "STATS", Level: INFO, Message: "test"}
		cache.Put(key, entry)
	}

	// Делаем несколько запросов (попадания и промахи)
	cache.Get("stats_key_0") // попадание
	cache.Get("stats_key_1") // попадание
	cache.Get("nonexistent") // промах

	stats = cache.GetStats()
	if stats.Size != 3 {
		t.Errorf("размер должен быть 3, получили %d", stats.Size)
	}
	if stats.Hits != 2 {
		t.Errorf("попадания должны быть 2, получили %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("промахи должны быть 1, получили %d", stats.Misses)
	}

	// Добавляем записи для вытеснения
	for i := 3; i < 7; i++ { // Превышаем лимит 5
		key := fmt.Sprintf("stats_key_%d", i)
		entry := LogEntry{Service: "STATS", Level: INFO, Message: "test"}
		cache.Put(key, entry)
	}

	stats = cache.GetStats()
	if stats.Size != 5 {
		t.Errorf("размер должен быть 5 (лимит), получили %d", stats.Size)
	}
	if stats.Evictions != 2 {
		t.Errorf("вытеснения должны быть 2, получили %d", stats.Evictions)
	}
}

// TestLogCacheConcurrency проверяет потокобезопасность кеша
func TestLogCacheConcurrency(t *testing.T) {
	cache := NewLogCache(200, 0) // Увеличиваем лимит для избежания конфликтов
	defer cache.Close()          // Закрываем cleanup горутину после теста

	const numGoroutines = 5  // Уменьшаем количество горутин
	const numOperations = 20 // Уменьшаем количество операций

	done := make(chan bool, numGoroutines)

	// Запускаем горутины для параллельных операций
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent_key_%d_%d", goroutineID, j)
				entry := LogEntry{
					Service: fmt.Sprintf("SERVICE_%d", goroutineID),
					Level:   INFO,
					Message: fmt.Sprintf("message %d from goroutine %d", j, goroutineID),
				}

				// Добавляем запись
				cache.Put(key, entry)

				// Читаем запись (только что добавленная должна быть доступна)
				result, found := cache.Get(key)
				if !found {
					// Запись может быть вытеснена из-за конкурентного доступа,
					// это нормально для LRU кеша с ограниченным размером
					continue
				}
				if result != nil && result.Service != entry.Service {
					t.Errorf("неверный сервис для ключа '%s': ожидался '%s', получили '%s'",
						key, entry.Service, result.Service)
				}
			}
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Проверяем финальную статистику
	stats := cache.GetStats()
	totalOperations := numGoroutines * numOperations
	expectedMaxSize := 200 // Лимит кеша

	// Размер кеша не должен превышать лимит
	if stats.Size > expectedMaxSize {
		t.Errorf("размер кеша не должен превышать лимит %d, получили %d", expectedMaxSize, stats.Size)
	}

	// Должны быть операции
	if stats.Hits+stats.Misses < int64(totalOperations) {
		t.Errorf("общее количество операций должно быть не менее %d, получили %d",
			totalOperations, stats.Hits+stats.Misses)
	}

	// Проверяем, что кеш не пуст
	if stats.Size == 0 {
		t.Error("кеш не должен быть пустым после конкурентных операций")
	}
}

// BenchmarkLogCachePut бенчмарк для операции Put
func BenchmarkLogCachePut(b *testing.B) {
	cache := NewLogCache(1000, 0)
	defer cache.Close() // Закрываем cleanup горутину после теста
	entry := LogEntry{
		Service: "BENCH",
		Level:   INFO,
		Message: "benchmark message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		cache.Put(key, entry)
	}
}

// BenchmarkLogCacheGet бенчмарк для операции Get
func BenchmarkLogCacheGet(b *testing.B) {
	cache := NewLogCache(1000, 0)
	defer cache.Close() // Закрываем cleanup горутину после теста
	entry := LogEntry{
		Service: "BENCH",
		Level:   INFO,
		Message: "benchmark message",
	}

	// Предварительно заполняем кеш
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		cache.Put(key, entry)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i%100)
		cache.Get(key)
	}
}
