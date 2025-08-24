// performance_test.go - Тесты производительности и памяти для embedded систем
//go:build logger_integration
// +build logger_integration

package logger

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkLoggerCreation бенчмарк создания логгера
func BenchmarkLoggerCreation(b *testing.B) {
	config := MockConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Создаем логгер (ожидаем ошибку подключения, но тестируем скорость создания)
		_, _ = New(config, []string{"BENCH"})
	}
}

// BenchmarkLogMessagePool бенчмарк пула сообщений
func BenchmarkLogMessagePool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := GetLogMessage()
		msg.Service = "BENCH"
		msg.Level = INFO
		msg.Message = "benchmark message"
		msg.Timestamp = time.Now()
		PutLogMessage(msg)
	}
}

// BenchmarkLogMessagePoolParallel параллельный бенчмарк пула сообщений
func BenchmarkLogMessagePoolParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg := GetLogMessage()
			msg.Service = "BENCH"
			msg.Level = INFO
			msg.Message = "parallel benchmark message"
			msg.Timestamp = time.Now()
			PutLogMessage(msg)
		}
	})
}

// BenchmarkServiceLoggerCaching бенчмарк кеширования ServiceLogger
func BenchmarkServiceLoggerCaching(b *testing.B) {
	mockClient := &MockLogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}

	services := []string{"API", "DNS", "VPN", "CONFIG", "MAIN"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := services[i%len(services)]
		_ = mockClient.SetService(service)
	}
}

// BenchmarkFilterOptionsValidation бенчмарк валидации FilterOptions
func BenchmarkFilterOptionsValidation(b *testing.B) {
	now := time.Now()
	past := now.Add(-time.Hour)
	infoLevel := INFO

	filter := FilterOptions{
		StartTime: &past,
		EndTime:   &now,
		Level:     &infoLevel,
		Service:   "BENCH",
		Limit:     100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filter.Validate()
	}
}

// TestMemoryUsageUnderLoad проверяет использование памяти под нагрузкой
func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тесты памяти в коротком режиме")
	}

	// Получаем начальную статистику памяти
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Создаем нагрузку
	const numOperations = 10000
	mockClient := &MockLogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}

	for i := 0; i < numOperations; i++ {
		// Тестируем пул сообщений
		msg := GetLogMessage()
		msg.Service = "MEMORY_TEST"
		msg.Level = INFO
		msg.Message = fmt.Sprintf("memory test message %d", i)
		msg.Timestamp = time.Now()
		PutLogMessage(msg)

		// Тестируем кеширование сервисов
		service := fmt.Sprintf("SERVICE_%d", i%10)
		_ = mockClient.SetService(service)

		// Тестируем парсинг уровней
		_, _ = ParseLevel("INFO")
	}

	// Принудительная сборка мусора
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Проверяем использование памяти
	allocDiff := m2.TotalAlloc - m1.TotalAlloc

	// Используем int64 для обработки возможных отрицательных значений
	heapDiff := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)

	t.Logf("Операций: %d", numOperations)
	t.Logf("Общие аллокации: %d байт", allocDiff)
	t.Logf("Heap аллокации: %d байт", heapDiff)
	t.Logf("Аллокаций на операцию: %.2f байт", float64(allocDiff)/float64(numOperations))

	// Проверяем, что память не растет неконтролируемо
	maxHeapPerOperation := int64(1024) // 1KB на операцию максимум

	// Проверяем только если есть рост памяти
	if heapDiff > 0 {
		heapPerOp := heapDiff / int64(numOperations)
		if heapPerOp > maxHeapPerOperation {
			t.Errorf("Слишком большое потребление памяти: %d байт на операцию", heapPerOp)
		}
	} else {
		// Память уменьшилась или не изменилась - это хорошо
		t.Logf("Память не увеличилась или уменьшилась: %d байт", heapDiff)
	}
}

// TestConcurrentMemoryUsage проверяет использование памяти при параллельной работе
func TestConcurrentMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тесты памяти в коротком режиме")
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	const numGoroutines = 50
	const operationsPerGoroutine = 200
	var wg sync.WaitGroup

	mockClient := &MockLogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Тестируем пул сообщений
				msg := GetLogMessage()
				msg.Service = fmt.Sprintf("GOROUTINE_%d", goroutineID)
				msg.Level = INFO
				msg.Message = fmt.Sprintf("concurrent message %d", j)
				msg.Timestamp = time.Now()
				PutLogMessage(msg)

				// Тестируем кеширование
				service := fmt.Sprintf("SERVICE_%d_%d", goroutineID, j%5)
				_ = mockClient.SetService(service)
			}
		}(i)
	}

	wg.Wait()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	totalOperations := numGoroutines * operationsPerGoroutine
	allocDiff := m2.TotalAlloc - m1.TotalAlloc
	heapDiff := m2.HeapAlloc - m1.HeapAlloc

	t.Logf("Горутин: %d", numGoroutines)
	t.Logf("Операций на горутину: %d", operationsPerGoroutine)
	t.Logf("Общих операций: %d", totalOperations)
	t.Logf("Общие аллокации: %d байт", allocDiff)
	t.Logf("Heap аллокации: %d байт", heapDiff)

	// Проверяем лимиты для embedded систем
	maxTotalMemory := uint64(DEFAULT_MAX_MEMORY) // 50MB из defaults.go
	if heapDiff > maxTotalMemory {
		t.Errorf("Превышен лимит памяти для embedded систем: %d > %d байт",
			heapDiff, maxTotalMemory)
	}
}

// TestMemoryLeaks проверяет утечки памяти
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тесты утечек памяти в коротком режиме")
	}

	// Делаем несколько циклов создания/уничтожения объектов
	const numCycles = 5
	const objectsPerCycle = 1000

	var memStats []runtime.MemStats

	for cycle := 0; cycle < numCycles; cycle++ {
		// Создаем объекты
		messages := make([]*LogMessage, objectsPerCycle)
		entries := make([]*LogEntry, objectsPerCycle)

		for i := 0; i < objectsPerCycle; i++ {
			messages[i] = GetLogMessage()
			messages[i].Service = "LEAK_TEST"
			messages[i].Level = INFO
			messages[i].Message = "leak test message"
			messages[i].Timestamp = time.Now()

			entries[i] = GetLogEntry()
			entries[i].Service = "LEAK_TEST"
			entries[i].Level = INFO
			entries[i].Message = "leak test entry"
			entries[i].Raw = "raw leak test entry"
		}

		// Возвращаем объекты в пул
		for i := 0; i < objectsPerCycle; i++ {
			PutLogMessage(messages[i])
			PutLogEntry(entries[i])
		}

		// Очищаем ссылки
		messages = nil
		entries = nil

		// Принудительная сборка мусора
		runtime.GC()
		runtime.GC() // Двойная сборка для надежности

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		memStats = append(memStats, m)

		t.Logf("Цикл %d: HeapAlloc=%d, HeapObjects=%d",
			cycle+1, m.HeapAlloc, m.HeapObjects)
	}

	// Проверяем, что память не растет значительно от цикла к циклу
	for i := 1; i < len(memStats); i++ {
		heapGrowth := int64(memStats[i].HeapAlloc) - int64(memStats[i-1].HeapAlloc)

		// Допускаем небольшой рост (до 10KB между циклами)
		maxGrowthPerCycle := int64(20 * 1024) // допускаем рост до 20KB

		if heapGrowth > maxGrowthPerCycle {
			t.Errorf("Подозрение на утечку памяти: рост %d байт между циклами %d и %d",
				heapGrowth, i, i+1)
		} else if heapGrowth < 0 {
			// Отрицательный рост означает, что GC освободил память - это хорошо
			t.Logf("Память уменьшилась на %d байт между циклами %d и %d (GC работает эффективно)",
				-heapGrowth, i, i+1)
		} else {
			t.Logf("Допустимый рост памяти: %d байт между циклами %d и %d",
				heapGrowth, i, i+1)
		}

		// Также проверяем количество объектов в куче
		objectGrowth := int64(memStats[i].HeapObjects) - int64(memStats[i-1].HeapObjects)
		if objectGrowth > 300 { // Допускаем рост до 300 объектов
			t.Errorf("Подозрение на утечку объектов: рост %d объектов между циклами %d и %d",
				objectGrowth, i, i+1)
		}
	}
}

// TestEmbeddedSystemLimits проверяет соответствие лимитам embedded систем
func TestEmbeddedSystemLimits(t *testing.T) {
	// Проверяем константы из defaults.go
	if DEFAULT_MAX_MEMORY > 100*1024*1024 { // 100MB
		t.Errorf("DEFAULT_MAX_MEMORY слишком большой для embedded систем: %d",
			DEFAULT_MAX_MEMORY)
	}

	if DEFAULT_MAX_CONNECTIONS > 50 {
		t.Errorf("DEFAULT_MAX_CONNECTIONS слишком большой для embedded систем: %d",
			DEFAULT_MAX_CONNECTIONS)
	}

	if DEFAULT_MAX_MESSAGE_SIZE > 8192 { // 8KB
		t.Errorf("DEFAULT_MAX_MESSAGE_SIZE слишком большой для embedded систем: %d",
			DEFAULT_MAX_MESSAGE_SIZE)
	}

	if DEFAULT_CACHE_SIZE > 1000 {
		t.Errorf("DEFAULT_CACHE_SIZE слишком большой для embedded систем: %d",
			DEFAULT_CACHE_SIZE)
	}

	if DEFAULT_RATE_LIMIT > 1000 { // 1000 msg/sec
		t.Errorf("DEFAULT_RATE_LIMIT слишком большой для embedded систем: %d",
			DEFAULT_RATE_LIMIT)
	}
}

// BenchmarkHighThroughput бенчмарк высокой пропускной способности
func BenchmarkHighThroughput(b *testing.B) {
	mockClient := &MockLogClient{}
	logger := &Logger{client: mockClient}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = logger.Info("high throughput message")
		}
	})
}

// BenchmarkServiceLoggerHighThroughput бенчмарк высокой пропускной способности для ServiceLogger
func BenchmarkServiceLoggerHighThroughput(b *testing.B) {
	mockClient := &MockLogClient{}
	serviceLogger := newServiceLogger(mockClient, "BENCH")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = serviceLogger.Info("service high throughput message")
		}
	})
}

// TestResourceConstraints проверяет работу в условиях ограниченных ресурсов
func TestResourceConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем тесты ограниченных ресурсов в коротком режиме")
	}

	// Устанавливаем лимит памяти для GC
	oldGOGC := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(oldGOGC)

	// Ограничиваем количество процессоров для эмуляции embedded системы
	runtime.GOMAXPROCS(2)

	const numOperations = 5000
	mockClient := &MockLogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}

	start := time.Now()

	// Выполняем операции в условиях ограниченных ресурсов
	for i := 0; i < numOperations; i++ {
		// Создаем сообщения
		msg := GetLogMessage()
		msg.Service = "CONSTRAINED"
		msg.Level = INFO
		msg.Message = fmt.Sprintf("constrained message %d", i)
		msg.Timestamp = time.Now()

		// Имитируем обработку
		_ = mockClient.sendMessage(msg.Service, msg.Level, msg.Message)

		PutLogMessage(msg)

		// Периодическая сборка мусора для эмуляции embedded системы
		if i%100 == 0 {
			runtime.GC()
		}
	}

	duration := time.Since(start)
	throughput := float64(numOperations) / duration.Seconds()

	t.Logf("Операций: %d", numOperations)
	t.Logf("Время: %v", duration)
	t.Logf("Пропускная способность: %.2f ops/sec", throughput)

	// Проверяем минимальную пропускную способность для embedded систем
	minThroughput := 100.0 // 100 операций в секунду минимум
	if throughput < minThroughput {
		t.Errorf("Пропускная способность слишком низкая: %.2f < %.2f ops/sec",
			throughput, minThroughput)
	}
}

// TestGoroutineLimits проверяет ограничения на количество горутин
func TestGoroutineLimits(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	const maxAdditionalGoroutines = 20 // Лимит для embedded систем
	var wg sync.WaitGroup

	// Создаем горутины для тестирования
	for i := 0; i < maxAdditionalGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			mockClient := &MockLogClient{}
			for j := 0; j < 100; j++ {
				_ = mockClient.sendMessage("GOROUTINE_TEST", INFO,
					fmt.Sprintf("goroutine %d message %d", id, j))
			}
		}(i)
	}

	wg.Wait()

	finalGoroutines := runtime.NumGoroutine()
	additionalGoroutines := finalGoroutines - initialGoroutines

	t.Logf("Начальные горутины: %d", initialGoroutines)
	t.Logf("Финальные горутины: %d", finalGoroutines)
	t.Logf("Дополнительные горутины: %d", additionalGoroutines)

	// Проверяем, что не создалось слишком много горутин
	if additionalGoroutines > 5 { // Допускаем небольшое количество служебных горутин
		t.Errorf("Создано слишком много дополнительных горутин: %d", additionalGoroutines)
	}
}
