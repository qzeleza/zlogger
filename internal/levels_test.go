// levels_test.go - Unit тесты для уровней логирования
package logger

import (
	"testing"
)

// TestLogLevelString проверяет строковое представление уровней
func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{PANIC, "PANIC"},
		{LogLevel(999), "UNKNOWN"}, // Неизвестный уровень
		{LogLevel(-1), "UNKNOWN"},  // Отрицательный уровень
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("ожидался '%s', получили '%s'", tt.expected, result)
			}
		})
	}
}

// TestLogLevelIsValid проверяет валидацию уровней
func TestLogLevelIsValid(t *testing.T) {
	tests := []struct {
		level LogLevel
		valid bool
	}{
		{DEBUG, true},
		{INFO, true},
		{WARN, true},
		{ERROR, true},
		{FATAL, true},
		{PANIC, true},
		{LogLevel(999), false}, // Неизвестный уровень
		{LogLevel(-1), false},  // Отрицательный уровень
		{LogLevel(6), false},   // Уровень за пределами диапазона
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			result := tt.level.IsValid()
			if result != tt.valid {
				t.Errorf("для уровня %v ожидалась валидность %v, получили %v", 
					tt.level, tt.valid, result)
			}
		})
	}
}

// TestParseLevel проверяет парсинг строковых уровней
func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
		wantErr  bool
	}{
		{"DEBUG", DEBUG, false},
		{"INFO", INFO, false},
		{"WARN", WARN, false},
		{"ERROR", ERROR, false},
		{"FATAL", FATAL, false},
		{"PANIC", PANIC, false},
		{"debug", DEBUG, false}, // Проверка нечувствительности к регистру
		{"Info", INFO, false},   // Смешанный регистр
		{"  WARN  ", WARN, false}, // Пробелы должны обрезаться
		{"UNKNOWN", INFO, true},   // Неизвестный уровень -> ошибка, возврат INFO
		{"", INFO, true},          // Пустая строка -> ошибка, возврат INFO
		{"123", INFO, true},       // Числовая строка -> ошибка, возврат INFO
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ParseLevel(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Error("ожидалась ошибка, но получили nil")
				}
				// При ошибке должен возвращаться INFO
				if level != INFO {
					t.Errorf("при ошибке ожидался уровень INFO, получили %v", level)
				}
			} else {
				if err != nil {
					t.Errorf("неожиданная ошибка: %v", err)
				}
				if level != tt.expected {
					t.Errorf("ожидался уровень %v, получили %v", tt.expected, level)
				}
			}
		})
	}
}

// TestLogLevelComparison проверяет сравнение уровней
func TestLogLevelComparison(t *testing.T) {
	tests := []struct {
		name   string
		level1 LogLevel
		level2 LogLevel
		less   bool
		equal  bool
	}{
		{"DEBUG < INFO", DEBUG, INFO, true, false},
		{"INFO < WARN", INFO, WARN, true, false},
		{"WARN < ERROR", WARN, ERROR, true, false},
		{"ERROR < FATAL", ERROR, FATAL, true, false},
		{"FATAL < PANIC", FATAL, PANIC, true, false},
		{"INFO == INFO", INFO, INFO, false, true},
		{"ERROR > WARN", ERROR, WARN, false, false},
		{"PANIC > DEBUG", PANIC, DEBUG, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			less := tt.level1 < tt.level2
			equal := tt.level1 == tt.level2
			
			if less != tt.less {
				t.Errorf("сравнение %v < %v: ожидалось %v, получили %v", 
					tt.level1, tt.level2, tt.less, less)
			}
			
			if equal != tt.equal {
				t.Errorf("сравнение %v == %v: ожидалось %v, получили %v", 
					tt.level1, tt.level2, tt.equal, equal)
			}
		})
	}
}

// TestLogLevelConstants проверяет правильность констант
func TestLogLevelConstants(t *testing.T) {
	// Проверяем, что уровни идут по порядку
	if DEBUG != 0 {
		t.Errorf("DEBUG должен быть 0, получили %d", DEBUG)
	}
	if INFO != 1 {
		t.Errorf("INFO должен быть 1, получили %d", INFO)
	}
	if WARN != 2 {
		t.Errorf("WARN должен быть 2, получили %d", WARN)
	}
	if ERROR != 3 {
		t.Errorf("ERROR должен быть 3, получили %d", ERROR)
	}
	if FATAL != 4 {
		t.Errorf("FATAL должен быть 4, получили %d", FATAL)
	}
	if PANIC != 5 {
		t.Errorf("PANIC должен быть 5, получили %d", PANIC)
	}
}

// TestLogLevelFiltering проверяет фильтрацию по уровням
func TestLogLevelFiltering(t *testing.T) {
	// Тестируем логику фильтрации: сообщения с уровнем >= установленного должны проходить
	tests := []struct {
		name         string
		setLevel     LogLevel
		messageLevel LogLevel
		shouldPass   bool
	}{
		{"DEBUG уровень пропускает все", DEBUG, DEBUG, true},
		{"DEBUG уровень пропускает INFO", DEBUG, INFO, true},
		{"DEBUG уровень пропускает ERROR", DEBUG, ERROR, true},
		{"INFO уровень блокирует DEBUG", INFO, DEBUG, false},
		{"INFO уровень пропускает INFO", INFO, INFO, true},
		{"INFO уровень пропускает WARN", INFO, WARN, true},
		{"ERROR уровень блокирует INFO", ERROR, INFO, false},
		{"ERROR уровень блокирует WARN", ERROR, WARN, false},
		{"ERROR уровень пропускает ERROR", ERROR, ERROR, true},
		{"ERROR уровень пропускает FATAL", ERROR, FATAL, true},
		{"PANIC уровень блокирует все кроме PANIC", PANIC, ERROR, false},
		{"PANIC уровень пропускает PANIC", PANIC, PANIC, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldPass := tt.messageLevel >= tt.setLevel
			
			if shouldPass != tt.shouldPass {
				t.Errorf("для уровня %v при установленном %v ожидалось %v, получили %v",
					tt.messageLevel, tt.setLevel, tt.shouldPass, shouldPass)
			}
		})
	}
}

// BenchmarkLogLevelString бенчмарк для метода String
func BenchmarkLogLevelString(b *testing.B) {
	level := INFO
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = level.String()
	}
}

// BenchmarkParseLevel бенчмарк для парсинга уровня
func BenchmarkParseLevel(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseLevel("INFO")
	}
}

// BenchmarkLogLevelComparison бенчмарк для сравнения уровней
func BenchmarkLogLevelComparison(b *testing.B) {
	level1 := INFO
	level2 := ERROR
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = level1 < level2
	}
}

// TestLogLevelThreadSafety проверяет потокобезопасность операций с уровнями
func TestLogLevelThreadSafety(t *testing.T) {
	const numGoroutines = 100
	const numOperations = 1000
	
	done := make(chan bool, numGoroutines)
	
	// Запускаем горутины для параллельного тестирования
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < numOperations; j++ {
				// Тестируем различные операции
				level := LogLevel(j % 6) // 0-5
				_ = level.String()
				_ = level.IsValid()
				_, _ = ParseLevel("INFO")
			}
		}()
	}
	
	// Ждем завершения всех горутин
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
