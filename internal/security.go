// security.go - Модуль безопасности для защиты от атак
package logger

import (
	"fmt"

	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SecurityConfig конфигурация безопасности
type SecurityConfig struct {
	MaxMessageLength    int            // Максимальная длина сообщения
	MaxServiceLength    int            // Максимальная длина имени сервиса
	AllowedServiceChars *regexp.Regexp // Разрешенные символы в именах сервисов
	RateLimitPerSecond  int            // Ограничение скорости сообщений в секунду
	BanDuration         time.Duration  // Длительность бана за превышение лимитов
}

// DefaultSecurityConfig возвращает конфигурацию безопасности по умолчанию
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		MaxMessageLength:    4096,                                // 4KB максимум
		MaxServiceLength:    32,                                  // 32 символа для имени сервиса
		AllowedServiceChars: regexp.MustCompile(`^[A-Z0-9_-]+$`), // Только заглавные буквы, цифры, _ и -
		RateLimitPerSecond:  100,                                 // 100 сообщений в секунду на клиента
		BanDuration:         time.Minute * 5,                     // Бан на 5 минут
	}
}

// RateLimiter ограничитель скорости для клиентов
type RateLimiter struct {
	clients map[string]*ClientInfo // Информация о клиентах
	mu      sync.RWMutex           // Мьютекс для безопасного доступа
	config  *SecurityConfig        // Конфигурация безопасности
	done    chan struct{}          // Канал для остановки cleanup горутины
}

// ClientInfo информация о клиенте для rate limiting
type ClientInfo struct {
	LastAccess    time.Time // Время последнего доступа
	MessageCount  int       // Количество сообщений в текущую секунду
	BannedUntil   time.Time // Время окончания бана
	TotalMessages int64     // Общее количество сообщений
}

// NewRateLimiter создает новый ограничитель скорости
func NewRateLimiter(config *SecurityConfig) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ClientInfo),
		config:  config,
		done:    make(chan struct{}),
	}

	// Запускаем фоновую очистку старых записей
	go rl.cleanup()

	// Регистрируем финалайзер для гарантированного закрытия горутины
	runtime.SetFinalizer(rl, func(r *RateLimiter) {
		r.Close()
	})

	return rl
}

// IsAllowed проверяет, разрешен ли доступ для клиента
func (rl *RateLimiter) IsAllowed(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[clientID]

	if !exists {
		// Новый клиент
		rl.clients[clientID] = &ClientInfo{
			LastAccess:    now,
			MessageCount:  1,
			TotalMessages: 1,
		}
		return true
	}

	// Проверяем, не забанен ли клиент
	if now.Before(client.BannedUntil) {
		return false
	}

	// Сбрасываем счетчик если прошла секунда
	if now.Sub(client.LastAccess) >= time.Second {
		client.MessageCount = 0
		client.LastAccess = now
	}

	client.MessageCount++
	client.TotalMessages++

	// Проверяем лимит
	if client.MessageCount > rl.config.RateLimitPerSecond {
		// Баним клиента
		client.BannedUntil = now.Add(rl.config.BanDuration)
		return false
	}

	return true
}

// cleanup очищает старые записи клиентов
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute * 10) // Чистим каждые 10 минут
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return // Завершаем горутину
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()

			for clientID, client := range rl.clients {
				// Удаляем клиентов, которые не активны более часа
				if now.Sub(client.LastAccess) > time.Hour {
					delete(rl.clients, clientID)
				}
			}

			rl.mu.Unlock()
		}
	}
}

// ValidateMessage проверяет корректность сообщения лога
func ValidateMessage(msg *LogMessage, config *SecurityConfig) error {
	// Проверяем, что параметры не nil
	if msg == nil {
		return fmt.Errorf("сообщение не может быть nil")
	}
	if config == nil {
		return fmt.Errorf("конфигурация не может быть nil")
	}

	// Проверяем длину сообщения
	if len(msg.Message) > config.MaxMessageLength {
		return fmt.Errorf("сообщение слишком длинное: %d > %d", len(msg.Message), config.MaxMessageLength)
	}

	// Проверяем длину имени сервиса
	if len(msg.Service) > config.MaxServiceLength {
		return fmt.Errorf("имя сервиса слишком длинное: %d > %d", len(msg.Service), config.MaxServiceLength)
	}

	// Проверяем символы в имени сервиса
	if !config.AllowedServiceChars.MatchString(msg.Service) {
		return fmt.Errorf("недопустимые символы в имени сервиса: %s", msg.Service)
	}

	// Проверяем уровень логирования
	if !msg.Level.IsValid() {
		return fmt.Errorf("недопустимый уровень логирования: %d", msg.Level)
	}

	// Проверяем на опасные символы в сообщении
	if strings.Contains(msg.Message, "\x00") {
		return fmt.Errorf("сообщение содержит null-байты")
	}

	return nil
}

// ValidateConfig проверяет безопасность конфигурации
func ValidateConfig(config *LoggingConfig) error {
	// Проверяем, что конфигурация не nil
	if config == nil {
		return fmt.Errorf("конфигурация не может быть nil")
	}

	// Проверяем пути файлов на безопасность
	if !filepath.IsAbs(config.LogFile) {
		return fmt.Errorf("путь к файлу лога должен быть абсолютным")
	}

	if !filepath.IsAbs(config.SocketPath) {
		return fmt.Errorf("путь к сокету должен быть абсолютным")
	}

	// Проверяем, что пути не содержат опасных символов
	dangerousChars := []string{"..", "\x00", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(config.LogFile, char) || strings.Contains(config.SocketPath, char) {
			return fmt.Errorf("пути содержат опасные символы")
		}
	}

	// Проверяем разумные лимиты
	if DEFAULT_MAX_CONNECTIONS > 1000 {
		return fmt.Errorf("слишком много одновременных подключений: %d", DEFAULT_MAX_CONNECTIONS)
	}

	if DEFAULT_MAX_MESSAGE_SIZE > 1024*1024 { // 1MB максимум
		return fmt.Errorf("слишком большой размер сообщения: %d", DEFAULT_MAX_MESSAGE_SIZE)
	}

	if config.BufferSize > 100000 {
		return fmt.Errorf("слишком большой размер буфера: %d", config.BufferSize)
	}

	return nil
}

// Close останавливает cleanup горутину RateLimiter
func (rl *RateLimiter) Close() {
	// Защита от двойного закрытия
	select {
	case <-rl.done:
		return
	default:
		close(rl.done)
	}
}
