// Package logger - экспортируемые функции для упрощенного интерфейса логирования
package logger

import "fmt"

// Глобальные функции для простого доступа к логированию без создания экземпляра

// Debug логирует сообщение с уровнем DEBUG
func Debug(message string) {
	fmt.Println("[DEBUG]", message)
}

// Info логирует сообщение с уровнем INFO
func Info(message string) {
	fmt.Println("[INFO]", message)
}

// Warn логирует сообщение с уровнем WARN
func Warn(message string) {
	fmt.Println("[WARN]", message)
}

// Error логирует сообщение с уровнем ERROR
func Error(message string) {
	fmt.Println("[ERROR]", message)
}

// Fatal логирует сообщение с уровнем FATAL
func Fatal(message string) {
	fmt.Println("[FATAL]", message)
}

// Debugf логирует форматированное сообщение с уровнем DEBUG
func Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

// Infof логирует форматированное сообщение с уровнем INFO
func Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

// Warnf логирует форматированное сообщение с уровнем WARN
func Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

// Errorf логирует форматированное сообщение с уровнем ERROR
func Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// Fatalf логирует форматированное сообщение с уровнем FATAL
func Fatalf(format string, args ...interface{}) {
	fmt.Printf("[FATAL] "+format+"\n", args...)
}