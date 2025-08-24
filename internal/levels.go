// levels.go - Уровни логирования с оптимизацией
package logger

import (
	"fmt"
	"strings"
)

// LogLevel уровни логирования с числовыми значениями для быстрого сравнения
type LogLevel int

const (
	DEBUG LogLevel = iota // 0 - Отладочная информация
	INFO                  // 1 - Информационные сообщения
	WARN                  // 2 - Предупреждения
	ERROR                 // 3 - Ошибки
	FATAL                 // 4 - Критические ошибки
	PANIC                 // 5 - Паника приложения
)

// Кешированные строковые представления для производительности
var levelNames = [...]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
	PANIC: "PANIC",
}

// Мапа для быстрого поиска уровня по строке
var levelValues = map[string]LogLevel{
	"DEBUG": DEBUG,
	"INFO":  INFO,
	"WARN":  WARN,
	"ERROR": ERROR,
	"FATAL": FATAL,
	"PANIC": PANIC,
}

// String возвращает строковое представление уровня (оптимизировано)
func (l LogLevel) String() string {
	if l >= 0 && int(l) < len(levelNames) {
		return levelNames[l]
	}
	return "UNKNOWN"
}

// IsValid проверяет валидность уровня логирования
func (l LogLevel) IsValid() bool {
	return l >= DEBUG && l <= PANIC
}

// ParseLevel парсит строковый уровень с улучшенной обработкой ошибок
func ParseLevel(level string) (LogLevel, error) {
	if l, ok := levelValues[strings.ToUpper(strings.TrimSpace(level))]; ok {
		return l, nil
	}
	return INFO, fmt.Errorf("неизвестный уровень логирования: %s", level)
}
