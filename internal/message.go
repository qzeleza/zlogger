// message.go - Структуры сообщений с пояснением использования форматов
package logger

import (
	"fmt"
	"sync"
	"time"
)

/*
ВАЖНО: Различие использования форматов данных в системе

1. JSON формат используется ТОЛЬКО для IPC (межпроцессного взаимодействия):
   - Протокол клиент-сервер через Unix сокеты
   - Структурированная передача команд и данных между процессами
   - Удобная сериализация/десериализация сложных структур
   - Не влияет на формат файла лога!

2. TXT формат используется для записи в ЛОГ ФАЙЛ:
   - Простой текстовый формат: [SERVICE] DATE TIME [LEVEL] "MESSAGE"
   - Легко читается человеком и скриптами
   - Минимальный overhead при записи на диск
   - Совместимость с утилитами типа grep, awk, tail

Пример:
- IPC (JSON): {"service":"DNS","level":1,"message":"Запрос обработан","timestamp":"..."}
- Лог файл (TXT): [DNS  ] 2024-01-15 14:30:23 [INFO ] "Запрос обработан"
*/

// LogMessage структура сообщения лога с оптимизацией памяти
type LogMessage struct {
	Service   string            `json:"service"`             // Название сервиса
	Level     LogLevel          `json:"level"`               // Уровень логирования
	Message   string            `json:"message"`             // Текст сообщения
	Timestamp time.Time         `json:"timestamp"`           // Время создания
	ClientID  string            `json:"client_id,omitempty"` // Идентификатор клиента
	Fields    map[string]string `json:"fields,omitempty"`    // Дополнительные поля для структурированного логирования
}

// LogEntry структура записи лога для чтения с кешированием
type LogEntry struct {
	Service   string    `json:"service"`   // Название сервиса
	Level     LogLevel  `json:"level"`     // Уровень логирования
	Message   string    `json:"message"`   // Текст сообщения
	Timestamp time.Time `json:"timestamp"` // Время создания
	Raw       string    `json:"raw"`       // Исходная строка лога
}

// FilterOptions опции фильтрации логов с валидацией
type FilterOptions struct {
	StartTime *time.Time `json:"start_time,omitempty"` // Начальное время фильтрации
	EndTime   *time.Time `json:"end_time,omitempty"`   // Конечное время фильтрации
	Level     *LogLevel  `json:"level,omitempty"`      // Фильтр по уровню
	Service   string     `json:"service,omitempty"`    // Фильтр по сервису
	Limit     int        `json:"limit,omitempty"`      // Лимит количества записей
}

// Validate проверяет корректность параметров фильтрации
func (f *FilterOptions) Validate() error {
	if f.StartTime != nil && f.EndTime != nil && f.StartTime.After(*f.EndTime) {
		return fmt.Errorf("начальное время не может быть больше конечного")
	}
	if f.Limit < 0 {
		return fmt.Errorf("лимит не может быть отрицательным")
	}
	if f.Limit > 10000 { // Защита от чрезмерных запросов
		return fmt.Errorf("лимит не может превышать 10000 записей")
	}
	return nil
}

// Протокол взаимодействия клиент-сервер
type ProtocolMessage struct {
	Type string      `json:"type"` // Тип сообщения
	Data interface{} `json:"data"` // Данные сообщения
}

// Константы типов сообщений протокола
const (
	MsgTypeLog         = "log"          // Сообщение лога
	MsgTypeGetEntries  = "get_entries"  // Запрос записей
	MsgTypeUpdateLevel = "update_level" // Обновление уровня
	MsgTypeShutdown    = "shutdown"     // Команда остановки
	MsgTypeResponse    = "response"     // Ответ сервера
	MsgTypeError       = "error"        // Ошибка
	MsgTypePing        = "ping"         // Проверка соединения
	MsgTypePong        = "pong"         // Ответ на ping
	MsgTypeCmd         = "cmd"          // Команда
	MsgTypeResp        = "resp"         // Ответ на команду
	MsgTypeAck         = "ack"          // Подтверждение
	MsgTypeSetLevel    = "set_level"    // Установка уровня логирования
	MsgTypeLogFile     = "log_file"     // Файл лога
	MsgTypeGetLogFile  = "get_log_file" // Получение файла лога
)

// Пул объектов для переиспользования (оптимизация памяти)
var (
	logMessagePool = sync.Pool{
		New: func() interface{} {
			return &LogMessage{}
		},
	}

	logEntryPool = sync.Pool{
		New: func() interface{} {
			return &LogEntry{}
		},
	}
)

// GetLogMessage получает объект LogMessage из пула
func GetLogMessage() *LogMessage {
	return logMessagePool.Get().(*LogMessage)
}

// PutLogMessage возвращает объект LogMessage в пул
func PutLogMessage(msg *LogMessage) {
	// Очищаем поля перед возвратом в пул
	msg.Service = ""
	msg.Message = ""
	msg.ClientID = ""
	msg.Timestamp = time.Time{}
	msg.Fields = nil // Очищаем дополнительные поля
	logMessagePool.Put(msg)
}

// GetLogEntry получает объект LogEntry из пула
func GetLogEntry() *LogEntry {
	return logEntryPool.Get().(*LogEntry)
}

// PutLogEntry возвращает объект LogEntry в пул
func PutLogEntry(entry *LogEntry) {
	// Очищаем поля перед возвратом в пул
	entry.Service = ""
	entry.Message = ""
	entry.Raw = ""
	entry.Timestamp = time.Time{}
	logEntryPool.Put(entry)
}
