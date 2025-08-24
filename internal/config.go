package logger

import "time"

// LoggingConfig определяет параметры системы логирования
// Оптимизирован для минимального потребления ресурсов
type LoggingConfig struct {
	Level            string        `yaml:"level"`             // Уровень логирования (debug, info, warn, error)
	LogFile          string        `yaml:"log_file"`          // Путь к лог файлу (новый формат)
	Dir              string        `yaml:"dir"`               // Путь к директории логов (старый формат для совместимости)
	SocketPath       string        `yaml:"socket_path"`       // Путь к Unix сокету для логов
	MaxFileSize      float64       `yaml:"max_file_size"`     // Максимальный размер лог-файла в MB
	MaxFiles         int           `yaml:"max_files"`         // Количество резервных копий лог-файлов
	MaxSize          int           `yaml:"max_size"`          // Старый формат: максимальный размер лог-файла в MB
	MaxBackups       int           `yaml:"max_backups"`       // Старый формат: количество резервных копий
	MaxAge           int           `yaml:"max_age"`           // Старый формат: максимальный возраст файлов в днях
	Compress         bool          `yaml:"compress"`          // Старый формат: сжимать старые логи
	Console          bool          `yaml:"console"`           // Старый формат: выводить в консоль
	BufferSize       int           `yaml:"buffer_size"`       // Размер буфера сообщений в памяти в строках
	FlushInterval    time.Duration `yaml:"flush_interval"`    // Интервал принудительного сброса буфера на диск
	Services         []string      `yaml:"services"`          // Список разрешенных сервисов для логирования
	RestrictServices bool          `yaml:"restrict_services"` // Ограничить логирование только указанными сервисами
}
