package logger

import (
	"fmt"
)

/**
 * processArgs обрабатывает вариативные аргументы и возвращает сообщение и поля
 * Поддерживает различные форматы входных данных
 *
 * @param args ...interface{} - вариативные аргументы для обработки
 * @return string - форматированное сообщение
 * @return map[string]string - карта дополнительных полей
 */
func processArgs(args ...interface{}) (string, map[string]string) {
	if len(args) == 0 {
		return "", nil
	}
	
	// По умолчанию первый аргумент - сообщение
	message := ""
	fields := make(map[string]string)
	
	// Проверяем первый аргумент
	switch v := args[0].(type) {
	case string:
		message = v
	case fmt.Stringer:
		message = v.String()
	case error:
		message = v.Error()
	default:
		message = fmt.Sprintf("%v", v)
	}
	
	// Проверяем остальные аргументы
	if len(args) > 1 {
		// Проверяем, есть ли карта полей
		if fieldsMap, ok := args[1].(map[string]string); ok {
			fields = fieldsMap
		} else if len(args) > 1 && len(args) % 2 == 1 {
			// Обрабатываем ключ-значение пары
			for i := 1; i < len(args); i += 2 {
				if i+1 < len(args) {
					key, ok := args[i].(string)
					if ok {
						fields[key] = fmt.Sprintf("%v", args[i+1])
					}
				}
			}
		} else if len(args) > 1 {
			// Форматированное сообщение
			if format, ok := args[0].(string); ok {
				message = fmt.Sprintf(format, args[1:]...)
			}
		}
	}
	
	return message, fields
}

/**
 * parseKeyValuePairs преобразует массив строк в map[string]string
 * Предполагается, что массив содержит пары ключ-значение
 * Если количество элементов нечетное, последний элемент игнорируется
 *
 * @param keyValues []string - массив строк с парами ключ-значение
 * @return map[string]string - карта ключ-значение
 */
func parseKeyValuePairs(keyValues []string) map[string]string {
	result := make(map[string]string)
	for i := 0; i < len(keyValues)-1; i += 2 {
		result[keyValues[i]] = keyValues[i+1]
	}
	return result
}
