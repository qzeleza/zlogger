// Package logger - экспортируемые функции для упрощенного интерфейса логирования
package logger

import "fmt"

// Глобальные функции для простого доступа к логированию без создания экземпляра

// Debug логирует сообщение с уровнем DEBUG с поддержкой различных типов аргументов
func Debug(args ...interface{}) {
	// Обрабатываем аргументы
	if len(args) == 0 {
		fmt.Println("[DEBUG] ")
		return
	}
	
	// Если первый аргумент - строка формата и есть дополнительные аргументы
	if format, ok := args[0].(string); ok && len(args) > 1 {
		fmt.Printf("[DEBUG] "+format+"\n", args[1:]...)
		return
	}
	
	// Если только один аргумент
	fmt.Println("[DEBUG]", args[0])
}

// Info логирует сообщение с уровнем INFO с поддержкой различных типов аргументов
func Info(args ...interface{}) {
	// Обрабатываем аргументы
	if len(args) == 0 {
		fmt.Println("[INFO] ")
		return
	}
	
	// Если первый аргумент - строка формата и есть дополнительные аргументы
	if format, ok := args[0].(string); ok && len(args) > 1 {
		fmt.Printf("[INFO] "+format+"\n", args[1:]...)
		return
	}
	
	// Если только один аргумент
	fmt.Println("[INFO]", args[0])
}

// Warn логирует сообщение с уровнем WARN с поддержкой различных типов аргументов
func Warn(args ...interface{}) {
	// Обрабатываем аргументы
	if len(args) == 0 {
		fmt.Println("[WARN] ")
		return
	}
	
	// Если первый аргумент - строка формата и есть дополнительные аргументы
	if format, ok := args[0].(string); ok && len(args) > 1 {
		fmt.Printf("[WARN] "+format+"\n", args[1:]...)
		return
	}
	
	// Если только один аргумент
	fmt.Println("[WARN]", args[0])
}

// Error логирует сообщение с уровнем ERROR с поддержкой различных типов аргументов
func Error(args ...interface{}) {
	// Обрабатываем аргументы
	if len(args) == 0 {
		fmt.Println("[ERROR] ")
		return
	}
	
	// Если первый аргумент - строка формата и есть дополнительные аргументы
	if format, ok := args[0].(string); ok && len(args) > 1 {
		fmt.Printf("[ERROR] "+format+"\n", args[1:]...)
		return
	}
	
	// Если только один аргумент
	fmt.Println("[ERROR]", args[0])
}

// Fatal логирует сообщение с уровнем FATAL с поддержкой различных типов аргументов
func Fatal(args ...interface{}) {
	// Обрабатываем аргументы
	if len(args) == 0 {
		fmt.Println("[FATAL] ")
		return
	}
	
	// Если первый аргумент - строка формата и есть дополнительные аргументы
	if format, ok := args[0].(string); ok && len(args) > 1 {
		fmt.Printf("[FATAL] "+format+"\n", args[1:]...)
		return
	}
	
	// Если только один аргумент
	fmt.Println("[FATAL]", args[0])
}

// Panic логирует сообщение с уровнем PANIC с поддержкой различных типов аргументов
func Panic(args ...interface{}) {
	// Обрабатываем аргументы
	if len(args) == 0 {
		fmt.Println("[PANIC] ")
		return
	}
	
	// Если первый аргумент - строка формата и есть дополнительные аргументы
	if format, ok := args[0].(string); ok && len(args) > 1 {
		fmt.Printf("[PANIC] "+format+"\n", args[1:]...)
		return
	}
	
	// Если только один аргумент
	fmt.Println("[PANIC]", args[0])
}