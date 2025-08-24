// message_test.go - Unit тесты для структур сообщений и memory pooling
package logger

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestLogMessage проверяет структуру LogMessage
func TestLogMessage(t *testing.T) {
	timestamp := time.Now()
	msg := LogMessage{
		Service:   "TEST",
		Level:     INFO,
		Message:   "test message",
		Timestamp: timestamp,
		ClientID:  "client-123",
	}

	// Проверяем поля
	if msg.Service != "TEST" {
		t.Errorf("ожидался сервис 'TEST', получили '%s'", msg.Service)
	}
	if msg.Level != INFO {
		t.Errorf("ожидался уровень INFO, получили %v", msg.Level)
	}
	if msg.Message != "test message" {
		t.Errorf("ожидалось сообщение 'test message', получили '%s'", msg.Message)
	}
	if !msg.Timestamp.Equal(timestamp) {
		t.Errorf("ожидалось время %v, получили %v", timestamp, msg.Timestamp)
	}
	if msg.ClientID != "client-123" {
		t.Errorf("ожидался ClientID 'client-123', получили '%s'", msg.ClientID)
	}
}

// TestLogMessageJSON проверяет JSON сериализацию/десериализацию LogMessage
func TestLogMessageJSON(t *testing.T) {
	original := LogMessage{
		Service:   "API",
		Level:     ERROR,
		Message:   "API error occurred",
		Timestamp: time.Date(2024, 1, 15, 14, 30, 23, 0, time.UTC),
		ClientID:  "api-client-456",
	}

	// Сериализация
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("ошибка сериализации: %v", err)
	}

	// Десериализация
	var decoded LogMessage
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("ошибка десериализации: %v", err)
	}

	// Проверяем соответствие
	if decoded.Service != original.Service {
		t.Errorf("сервис не совпадает: ожидался '%s', получили '%s'", 
			original.Service, decoded.Service)
	}
	if decoded.Level != original.Level {
		t.Errorf("уровень не совпадает: ожидался %v, получили %v", 
			original.Level, decoded.Level)
	}
	if decoded.Message != original.Message {
		t.Errorf("сообщение не совпадает: ожидалось '%s', получили '%s'", 
			original.Message, decoded.Message)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("время не совпадает: ожидалось %v, получили %v", 
			original.Timestamp, decoded.Timestamp)
	}
	if decoded.ClientID != original.ClientID {
		t.Errorf("ClientID не совпадает: ожидался '%s', получили '%s'", 
			original.ClientID, decoded.ClientID)
	}
}

// TestLogEntry проверяет структуру LogEntry
func TestLogEntry(t *testing.T) {
	timestamp := time.Now()
	entry := LogEntry{
		Service:   "DNS",
		Level:     WARN,
		Message:   "DNS таймаут",
		Timestamp: timestamp,
		Raw:       "[DNS  ] 2024-01-15 14:30:23 [WARN ] \"DNS таймаут\"",
	}

	// Проверяем поля
	if entry.Service != "DNS" {
		t.Errorf("ожидался сервис 'DNS', получили '%s'", entry.Service)
	}
	if entry.Level != WARN {
		t.Errorf("ожидался уровень WARN, получили %v", entry.Level)
	}
	if entry.Message != "DNS таймаут" {
		t.Errorf("ожидалось сообщение 'DNS таймаут', получили '%s'", entry.Message)
	}
	if !entry.Timestamp.Equal(timestamp) {
		t.Errorf("ожидалось время %v, получили %v", timestamp, entry.Timestamp)
	}
	if entry.Raw == "" {
		t.Error("Raw поле не должно быть пустым")
	}
}

// TestFilterOptionsValidate проверяет валидацию FilterOptions
func TestFilterOptionsValidate(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	infoLevel := INFO

	tests := []struct {
		name    string
		filter  FilterOptions
		wantErr bool
	}{
		{
			name: "валидный фильтр",
			filter: FilterOptions{
				StartTime: &past,
				EndTime:   &future,
				Level:     &infoLevel,
				Service:   "TEST",
				Limit:     100,
			},
			wantErr: false,
		},
		{
			name: "пустой фильтр",
			filter: FilterOptions{},
			wantErr: false,
		},
		{
			name: "начальное время больше конечного",
			filter: FilterOptions{
				StartTime: &future,
				EndTime:   &past,
			},
			wantErr: true,
		},
		{
			name: "отрицательный лимит",
			filter: FilterOptions{
				Limit: -1,
			},
			wantErr: true,
		},
		{
			name: "лимит превышает максимум",
			filter: FilterOptions{
				Limit: 20000,
			},
			wantErr: true,
		},
		{
			name: "максимальный допустимый лимит",
			filter: FilterOptions{
				Limit: 10000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			
			if tt.wantErr && err == nil {
				t.Error("ожидалась ошибка, но получили nil")
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
			}
		})
	}
}

// TestProtocolMessage проверяет структуру ProtocolMessage
func TestProtocolMessage(t *testing.T) {
	data := map[string]interface{}{
		"test": "value",
		"num":  123,
	}

	msg := ProtocolMessage{
		Type: MsgTypeLog,
		Data: data,
	}

	if msg.Type != MsgTypeLog {
		t.Errorf("ожидался тип '%s', получили '%s'", MsgTypeLog, msg.Type)
	}

	// Проверяем данные
	if dataMap, ok := msg.Data.(map[string]interface{}); ok {
		if dataMap["test"] != "value" {
			t.Errorf("ожидалось значение 'value', получили '%v'", dataMap["test"])
		}
		if dataMap["num"] != 123 {
			t.Errorf("ожидалось значение 123, получили '%v'", dataMap["num"])
		}
	} else {
		t.Error("данные должны быть map[string]interface{}")
	}
}

// TestMessageConstants проверяет константы типов сообщений
func TestMessageConstants(t *testing.T) {
	constants := map[string]string{
		MsgTypeLog:         "log",
		MsgTypeGetEntries:  "get_entries",
		MsgTypeUpdateLevel: "update_level",
		MsgTypeShutdown:    "shutdown",
		MsgTypeResponse:    "response",
		MsgTypeError:       "error",
		MsgTypePing:        "ping",
		MsgTypePong:        "pong",
	}

	for constant, expected := range constants {
		if constant != expected {
			t.Errorf("константа %s должна быть '%s', получили '%s'", 
				constant, expected, constant)
		}
	}
}

// TestLogMessagePool проверяет работу пула объектов LogMessage
func TestLogMessagePool(t *testing.T) {
	// Получаем объект из пула
	msg1 := GetLogMessage()
	if msg1 == nil {
		t.Fatal("GetLogMessage() не должен возвращать nil")
	}

	// Заполняем объект
	msg1.Service = "TEST"
	msg1.Level = INFO
	msg1.Message = "test message"
	msg1.Timestamp = time.Now()
	msg1.ClientID = "client-123"

	// Возвращаем в пул
	PutLogMessage(msg1)

	// Проверяем, что поля очищены
	if msg1.Service != "" {
		t.Error("Service должен быть очищен после возврата в пул")
	}
	if msg1.Message != "" {
		t.Error("Message должен быть очищен после возврата в пул")
	}
	if msg1.ClientID != "" {
		t.Error("ClientID должен быть очищен после возврата в пул")
	}
	if !msg1.Timestamp.IsZero() {
		t.Error("Timestamp должен быть очищен после возврата в пул")
	}

	// Получаем новый объект (может быть тот же самый из пула)
	msg2 := GetLogMessage()
	if msg2 == nil {
		t.Fatal("GetLogMessage() не должен возвращать nil")
	}

	// Проверяем, что объект чистый
	if msg2.Service != "" {
		t.Error("новый объект из пула должен иметь пустой Service")
	}
	if msg2.Message != "" {
		t.Error("новый объект из пула должен иметь пустой Message")
	}
}

// TestLogEntryPool проверяет работу пула объектов LogEntry
func TestLogEntryPool(t *testing.T) {
	// Получаем объект из пула
	entry1 := GetLogEntry()
	if entry1 == nil {
		t.Fatal("GetLogEntry() не должен возвращать nil")
	}

	// Заполняем объект
	entry1.Service = "API"
	entry1.Level = ERROR
	entry1.Message = "API error"
	entry1.Timestamp = time.Now()
	entry1.Raw = "[API] ERROR API error"

	// Возвращаем в пул
	PutLogEntry(entry1)

	// Проверяем, что поля очищены
	if entry1.Service != "" {
		t.Error("Service должен быть очищен после возврата в пул")
	}
	if entry1.Message != "" {
		t.Error("Message должен быть очищен после возврата в пул")
	}
	if entry1.Raw != "" {
		t.Error("Raw должен быть очищен после возврата в пул")
	}
	if !entry1.Timestamp.IsZero() {
		t.Error("Timestamp должен быть очищен после возврата в пул")
	}

	// Получаем новый объект
	entry2 := GetLogEntry()
	if entry2 == nil {
		t.Fatal("GetLogEntry() не должен возвращать nil")
	}

	// Проверяем, что объект чистый
	if entry2.Service != "" {
		t.Error("новый объект из пула должен иметь пустой Service")
	}
	if entry2.Raw != "" {
		t.Error("новый объект из пула должен иметь пустой Raw")
	}
}

// TestPoolConcurrency проверяет потокобезопасность пулов
func TestPoolConcurrency(t *testing.T) {
	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Для LogMessage и LogEntry

	// Тестируем LogMessage пул
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				msg := GetLogMessage()
				msg.Service = "TEST"
				msg.Message = "concurrent test"
				PutLogMessage(msg)
			}
		}()
	}

	// Тестируем LogEntry пул
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				entry := GetLogEntry()
				entry.Service = "TEST"
				entry.Raw = "concurrent test"
				PutLogEntry(entry)
			}
		}()
	}

	wg.Wait()
}

// BenchmarkGetLogMessage бенчмарк для получения LogMessage из пула
func BenchmarkGetLogMessage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := GetLogMessage()
		PutLogMessage(msg)
	}
}

// BenchmarkGetLogEntry бенчмарк для получения LogEntry из пула
func BenchmarkGetLogEntry(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := GetLogEntry()
		PutLogEntry(entry)
	}
}

// BenchmarkLogMessageJSON бенчмарк для JSON сериализации LogMessage
func BenchmarkLogMessageJSON(b *testing.B) {
	msg := LogMessage{
		Service:   "BENCH",
		Level:     INFO,
		Message:   "benchmark message",
		Timestamp: time.Now(),
		ClientID:  "bench-client",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestMemoryUsage проверяет использование памяти структурами
func TestMemoryUsage(t *testing.T) {
	// Создаем большое количество объектов для проверки утечек памяти
	const numObjects = 10000

	// Тестируем LogMessage
	for i := 0; i < numObjects; i++ {
		msg := GetLogMessage()
		msg.Service = "MEMORY_TEST"
		msg.Level = INFO
		msg.Message = "memory usage test message"
		msg.Timestamp = time.Now()
		PutLogMessage(msg)
	}

	// Тестируем LogEntry
	for i := 0; i < numObjects; i++ {
		entry := GetLogEntry()
		entry.Service = "MEMORY_TEST"
		entry.Level = INFO
		entry.Message = "memory usage test message"
		entry.Raw = "[MEMORY_TEST] INFO memory usage test message"
		PutLogEntry(entry)
	}

	// Если тест проходит без panic или out of memory, то все в порядке
	t.Log("Memory usage test completed successfully")
}
