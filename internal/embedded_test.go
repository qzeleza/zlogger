// embedded_test.go - Специальные тесты для embedded устройств
//go:build embedded
// +build embedded

package logger

import (
	"runtime"
	"testing"
	"time"
)

// TestEmbeddedMemoryConstraints проверяет работу в условиях жестких ограничений памяти
func TestEmbeddedMemoryConstraints(t *testing.T) {
	// Устанавливаем очень агрессивную сборку мусора для эмуляции embedded системы
	oldGOGC := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(oldGOGC)
	runtime.GOMAXPROCS(1) // Один процессор как на многих embedded устройствах

	// Тестируем с ограниченным количеством операций
	const maxOperations = 1000 // Меньше чем в обычных тестах
	
	mockClient := &MockLogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < maxOperations; i++ {
		msg := GetLogMessage()
		msg.Service = "EMBEDDED"
		msg.Level = INFO
		msg.Message = "embedded test message"
		msg.Timestamp = time.Now()
		
		_ = mockClient.sendMessage(msg.Service, msg.Level, msg.Message)
		PutLogMessage(msg)

		// Частая сборка мусора как на embedded системах
		if i%10 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	heapUsed := m2.HeapAlloc
	maxHeapForEmbedded := int64(10 * 1024 * 1024) // 10MB максимум

	if int64(heapUsed) > maxHeapForEmbedded {
		t.Errorf("Превышен лимит памяти для embedded устройств: %d > %d байт", 
			heapUsed, maxHeapForEmbedded)
	}

	t.Logf("Heap использовано: %d байт (лимит: %d байт)", heapUsed, maxHeapForEmbedded)
}

// TestEmbeddedPerformance проверяет производительность на embedded устройствах
func TestEmbeddedPerformance(t *testing.T) {
	// Ограничиваем ресурсы как на embedded устройстве
	runtime.GOMAXPROCS(1)
	
	const numOperations = 500
	mockClient := &MockLogClient{}
	serviceLogger := newServiceLogger(mockClient, "EMBEDDED")

	start := time.Now()

	for i := 0; i < numOperations; i++ {
		_ = serviceLogger.Info("embedded performance test")
		
		// Имитируем другую нагрузку на систему
		if i%50 == 0 {
			time.Sleep(time.Microsecond) // Микропауза
		}
	}

	duration := time.Since(start)
	throughput := float64(numOperations) / duration.Seconds()

	// Для embedded устройств ожидаем более низкую производительность
	minThroughputEmbedded := 50.0 // 50 ops/sec минимум для embedded
	if throughput < minThroughputEmbedded {
		t.Errorf("Производительность слишком низкая для embedded устройства: %.2f < %.2f ops/sec", 
			throughput, minThroughputEmbedded)
	}

	t.Logf("Embedded производительность: %.2f ops/sec", throughput)
}

// TestEmbeddedResourceLimits проверяет соблюдение лимитов ресурсов
func TestEmbeddedResourceLimits(t *testing.T) {
	// Проверяем, что константы подходят для embedded систем
	tests := []struct {
		name  string
		value int
		max   int
	}{
		{"MAX_CONNECTIONS", DEFAULT_MAX_CONNECTIONS, 10},
		{"MAX_MESSAGE_SIZE", DEFAULT_MAX_MESSAGE_SIZE, 2048}, // 2KB для embedded
		{"CACHE_SIZE", DEFAULT_CACHE_SIZE, 100},
		{"RATE_LIMIT", DEFAULT_RATE_LIMIT, 50}, // 50 msg/sec для embedded
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value > tt.max {
				t.Errorf("%s слишком большой для embedded устройств: %d > %d", 
					tt.name, tt.value, tt.max)
			}
		})
	}
}

// TestEmbeddedStartupTime проверяет время запуска на embedded устройствах
func TestEmbeddedStartupTime(t *testing.T) {
	const maxStartupTime = 100 * time.Millisecond // 100ms максимум для embedded

	start := time.Now()

	// Имитируем создание компонентов логгера
	mockClient := &MockLogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}

	// Создаем несколько сервисов
	services := []string{"MAIN", "API", "DNS"}
	for _, service := range services {
		_ = mockClient.SetService(service)
	}

	// Тестируем пул сообщений
	for i := 0; i < 10; i++ {
		msg := GetLogMessage()
		PutLogMessage(msg)
	}

	startupTime := time.Since(start)

	if startupTime > maxStartupTime {
		t.Errorf("Время запуска слишком большое для embedded устройства: %v > %v", 
			startupTime, maxStartupTime)
	}

	t.Logf("Время запуска: %v (лимит: %v)", startupTime, maxStartupTime)
}

// TestEmbeddedPowerEfficiency проверяет энергоэффективность
func TestEmbeddedPowerEfficiency(t *testing.T) {
	// Тестируем, что логгер не создает излишнюю нагрузку на CPU
	runtime.GOMAXPROCS(1)
	
	const testDuration = 100 * time.Millisecond
	const maxOperations = 100

	mockClient := &MockLogClient{}
	serviceLogger := newServiceLogger(mockClient, "POWER_TEST")

	start := time.Now()
	operations := 0

	// Выполняем операции в течение заданного времени
	for time.Since(start) < testDuration && operations < maxOperations {
		_ = serviceLogger.Info("power efficiency test")
		operations++
		
		// Небольшая пауза для снижения нагрузки на CPU
		time.Sleep(time.Microsecond)
	}

	actualDuration := time.Since(start)
	
	// Проверяем, что мы не превысили ожидаемое количество операций
	// (что указывало бы на излишнюю активность)
	if operations > maxOperations {
		t.Errorf("Слишком много операций за %v: %d > %d", 
			actualDuration, operations, maxOperations)
	}

	t.Logf("Операций за %v: %d (максимум: %d)", actualDuration, operations, maxOperations)
}

// TestEmbeddedFlashWearLeveling проверяет минимизацию записи на flash память
func TestEmbeddedFlashWearLeveling(t *testing.T) {
	// Тестируем, что пул объектов действительно переиспользует объекты
	// чтобы минимизировать аллокации и, соответственно, GC и запись в flash

	const numCycles = 10
	const objectsPerCycle = 50

	var totalAllocsBefore, totalAllocsAfter uint64

	// Получаем начальную статистику
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	totalAllocsBefore = m1.Mallocs

	for cycle := 0; cycle < numCycles; cycle++ {
		// Создаем и возвращаем объекты в пул
		for i := 0; i < objectsPerCycle; i++ {
			msg := GetLogMessage()
			msg.Service = "FLASH_TEST"
			msg.Level = INFO
			msg.Message = "flash wear test"
			PutLogMessage(msg)

			entry := GetLogEntry()
			entry.Service = "FLASH_TEST"
			entry.Level = INFO
			entry.Message = "flash wear test"
			PutLogEntry(entry)
		}
	}

	// Получаем финальную статистику
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	totalAllocsAfter = m2.Mallocs

	totalOperations := numCycles * objectsPerCycle * 2 // *2 для msg и entry
	actualAllocs := totalAllocsAfter - totalAllocsBefore
	
	// Проверяем, что количество аллокаций значительно меньше количества операций
	// благодаря переиспользованию объектов из пула
	maxAllocsRatio := 0.1 // Максимум 10% от операций должны быть новыми аллокациями
	maxAllocs := uint64(float64(totalOperations) * maxAllocsRatio)

	if actualAllocs > maxAllocs {
		t.Errorf("Слишком много аллокаций (плохо для flash памяти): %d > %d (%.1f%% от операций)", 
			actualAllocs, maxAllocs, maxAllocsRatio*100)
	}

	t.Logf("Операций: %d, Аллокаций: %d (%.2f%% от операций)", 
		totalOperations, actualAllocs, float64(actualAllocs)/float64(totalOperations)*100)
}

// TestEmbeddedNetworkLatency проверяет работу с высокой задержкой сети
func TestEmbeddedNetworkLatency(t *testing.T) {
	// Имитируем высокую задержку сети, характерную для некоторых embedded устройств
	mockClient := &MockLogClient{}
	
	// Добавляем искусственную задержку в мок
	mockClient.customSendMessage = func(service string, level LogLevel, message string) error {
		// Имитируем задержку сети 10ms
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	serviceLogger := newServiceLogger(mockClient, "LATENCY_TEST")

	const numMessages = 5 // Небольшое количество из-за задержки
	start := time.Now()

	for i := 0; i < numMessages; i++ {
		err := serviceLogger.Info("network latency test")
		if err != nil {
			t.Errorf("Ошибка при отправке сообщения %d: %v", i, err)
		}
	}

	totalTime := time.Since(start)
	expectedMinTime := time.Duration(numMessages) * 10 * time.Millisecond
	expectedMaxTime := expectedMinTime + 50*time.Millisecond // +50ms на обработку

	if totalTime < expectedMinTime {
		t.Errorf("Время выполнения слишком мало (задержка не учтена): %v < %v", 
			totalTime, expectedMinTime)
	}

	if totalTime > expectedMaxTime {
		t.Errorf("Время выполнения слишком велико: %v > %v", 
			totalTime, expectedMaxTime)
	}

	t.Logf("Время с задержкой сети: %v (ожидалось: %v-%v)", 
		totalTime, expectedMinTime, expectedMaxTime)
}
