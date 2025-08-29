// message_utils.go - Вспомогательные функции для обработки сообщений логгера
package logger

import (
	"fmt"
)

// processLogArgs обрабатывает различные типы аргументов для методов логирования
// и возвращает сообщение и дополнительные поля
//
// Поддерживаемые форматы:
// - Debug(message string) - простое сообщение
// - Debug(message string, fields map[string]string) - сообщение с полями в виде карты
// - Debug(format string, args ...interface{}) - форматированное сообщение
// - Debug(message string, keyValues ...string) - сообщение с полями в виде пар ключ-значение
//
// @param args - аргументы для обработки
// @return message - итоговое сообщение
// @return fields - дополнительные поля
// @return err - ошибка, если аргументы некорректны
func processLogArgs(args ...interface{}) (message string, fields map[string]string, err error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("отсутствуют аргументы")
	}

	// Проверяем первый аргумент, который должен быть строкой
	firstArg, ok := args[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("первый аргумент должен быть строкой")
	}

	// Если только один аргумент, возвращаем его как сообщение
	if len(args) == 1 {
		return firstArg, nil, nil
	}

	// Проверяем второй аргумент
	switch secondArg := args[1].(type) {
	case map[string]string:
		// Формат: Debug(message string, fields map[string]string)
		return firstArg, secondArg, nil

	case string:
		// Проверяем, является ли это форматированным сообщением или парами ключ-значение
		if len(args) > 2 {
			// Проверяем, все ли остальные аргументы - строки
			allStrings := true
			for i := 2; i < len(args); i++ {
				if _, ok := args[i].(string); !ok {
					allStrings = false
					break
				}
			}

			if allStrings {
				// Формат: Debug(message string, keyValues ...string)
				// Преобразуем в map[string]string
				strArgs := make([]string, 0, len(args)-1)
				for i := 1; i < len(args); i++ {
					strArgs = append(strArgs, args[i].(string))
				}
				fields = parseKeyValuePairs(strArgs)
				return firstArg, fields, nil
			} else {
				// Формат: Debug(format string, args ...interface{})
				// Форматируем сообщение
				return fmt.Sprintf(firstArg, args[1:]...), nil, nil
			}
		} else {
			// Формат: Debug(message string, singleKeyOrValue string)
			// Одиночная строка как второй аргумент - считаем это ключом без значения
			fields = make(map[string]string)
			fields[secondArg] = ""
			return firstArg, fields, nil
		}

	default:
		// Формат: Debug(format string, args ...interface{})
		// Форматируем сообщение
		formatArgs := make([]interface{}, 0, len(args)-1)
		for i := 1; i < len(args); i++ {
			formatArgs = append(formatArgs, args[i])
		}
		return fmt.Sprintf(firstArg, formatArgs...), nil, nil
	}
}

// isFormatString проверяет, является ли строка форматной строкой
func isFormatString(s string) bool {
	// Простая проверка на наличие спецификаторов формата
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+1 < len(s) && s[i+1] != '%' {
			return true
		}
	}
	return false
}

// parseKeyValueArgs преобразует массив аргументов в сообщение и поля
// Первый аргумент - сообщение, остальные - пары ключ-значение
func parseKeyValueArgs(args []interface{}) (string, map[string]string) {
	if len(args) == 0 {
		return "", nil
	}

	// Первый аргумент - сообщение
	message, ok := args[0].(string)
	if !ok {
		message = fmt.Sprintf("%v", args[0])
	}

	// Если только один аргумент, возвращаем его как сообщение
	if len(args) == 1 {
		return message, nil
	}

	// Преобразуем остальные аргументы в пары ключ-значение
	fields := make(map[string]string)
	for i := 1; i < len(args); i += 2 {
		key := fmt.Sprintf("%v", args[i])
		var value string
		if i+1 < len(args) {
			value = fmt.Sprintf("%v", args[i+1])
		}
		fields[key] = value
	}

	return message, fields
}
