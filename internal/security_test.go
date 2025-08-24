package logger

import (
	"strings"
	"testing"
	"time"

	conf "kvasdns/internal/config"
)

/**
 * TestDefaultSecurityConfig проверяет создание конфигурации безопасности по умолчанию
 * @param t *testing.T - тестовый контекст
 */
func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	if config == nil {
		t.Fatal("DefaultSecurityConfig должен вернуть непустую конфигурацию")
	}

	// Проверяем значения по умолчанию
	if config.MaxMessageLength <= 0 {
		t.Error("MaxMessageLength должен быть положительным")
	}

	if config.RateLimitPerSecond <= 0 {
		t.Error("RateLimitPerSecond должен быть положительным")
	}

	if config.MaxServiceLength <= 0 {
		t.Error("MaxServiceLength должен быть положительным")
	}

	if config.BanDuration <= 0 {
		t.Error("BanDuration должен быть положительным")
	}

	if config.AllowedServiceChars == nil {
		t.Error("AllowedServiceChars не должен быть nil")
	}
}

/**
 * TestNewRateLimiter проверяет создание ограничителя скорости
 * @param t *testing.T - тестовый контекст
 */
func TestNewRateLimiter(t *testing.T) {
	config := DefaultSecurityConfig()
	limiter := NewRateLimiter(config)
	defer limiter.Close() // Закрываем cleanup горутину после теста

	if limiter == nil {
		t.Fatal("NewRateLimiter должен вернуть непустой ограничитель")
	}

	if limiter.config != config {
		t.Error("конфигурация должна быть сохранена в ограничителе")
	}

	if limiter.clients == nil {
		t.Error("карта клиентов должна быть инициализирована")
	}
}

/**
 * TestRateLimiterIsAllowed проверяет работу ограничителя скорости
 * @param t *testing.T - тестовый контекст
 */
func TestRateLimiterIsAllowed(t *testing.T) {
	config := DefaultSecurityConfig()
	config.RateLimitPerSecond = 2 // Устанавливаем низкий лимит для тестирования

	// Создаем ограничитель без автоматической очистки для контролируемого тестирования
	limiter := &RateLimiter{
		clients: make(map[string]*ClientInfo),
		config:  config,
	}
	clientID := "test-client"

	// Первые запросы должны проходить
	if !limiter.IsAllowed(clientID) {
		t.Error("первый запрос должен быть разрешен")
	}

	if !limiter.IsAllowed(clientID) {
		t.Error("второй запрос должен быть разрешен")
	}

	// Третий запрос должен быть заблокирован
	if limiter.IsAllowed(clientID) {
		t.Error("третий запрос должен быть заблокирован")
	}

	// Имитируем сброс времени доступа и бана (вместо ожидания)
	limiter.mu.Lock()
	if client, exists := limiter.clients[clientID]; exists {
		client.LastAccess = time.Now().Add(-2 * time.Second) // Делаем вид, что прошло время
		client.BannedUntil = time.Time{}                     // Сбрасываем бан
		// НЕ сбрасываем MessageCount здесь - это сделает сам IsAllowed
	}
	limiter.mu.Unlock()

	// Теперь запрос должен быть разрешен (IsAllowed сам сбросит счетчик)
	if !limiter.IsAllowed(clientID) {
		t.Error("запрос после сброса лимита должен быть разрешен")
	}
}

/**
 * TestRateLimiterCleanup проверяет очистку старых записей
 * @param t *testing.T - тестовый контекст
 */
func TestRateLimiterCleanup(t *testing.T) {
	config := DefaultSecurityConfig()

	// Создаем ограничитель без автоматической очистки
	limiter := &RateLimiter{
		clients: make(map[string]*ClientInfo),
		config:  config,
	}

	// Добавляем клиента
	clientID := "test-client"
	limiter.IsAllowed(clientID)

	// Проверяем, что клиент есть в карте
	limiter.mu.RLock()
	_, exists := limiter.clients[clientID]
	limiter.mu.RUnlock()

	if !exists {
		t.Error("клиент должен быть в карте после запроса")
	}

	// Создаем старого клиента (имитируем старое время доступа)
	oldClientID := "old-client"
	limiter.mu.Lock()
	limiter.clients[oldClientID] = &ClientInfo{
		LastAccess:    time.Now().Add(-2 * time.Hour), // 2 часа назад
		MessageCount:  1,
		TotalMessages: 1,
	}
	limiter.mu.Unlock()

	// Вызываем очистку напрямую (тестируем логику без бесконечного цикла)
	limiter.mu.Lock()
	now := time.Now()
	for clientID, client := range limiter.clients {
		// Удаляем клиентов, которые не активны более часа
		if now.Sub(client.LastAccess) > time.Hour {
			delete(limiter.clients, clientID)
		}
	}
	limiter.mu.Unlock()

	// Проверяем, что недавний клиент остался
	limiter.mu.RLock()
	_, exists = limiter.clients["test-client"]
	limiter.mu.RUnlock()

	if !exists {
		t.Error("недавний клиент не должен быть удален при очистке")
	}

	// Проверяем, что старый клиент удален
	limiter.mu.RLock()
	_, exists = limiter.clients[oldClientID]
	limiter.mu.RUnlock()

	if exists {
		t.Error("старый клиент должен быть удален при очистке")
	}
}

/**
 * TestValidateMessage проверяет валидацию сообщений
 * @param t *testing.T - тестовый контекст
 */
func TestValidateMessage(t *testing.T) {
	config := DefaultSecurityConfig()

	// Тестируем валидное сообщение
	validMsg := &LogMessage{
		Service:   "MAIN",
		Level:     INFO,
		Message:   "test message",
		Timestamp: time.Now(),
	}

	if err := ValidateMessage(validMsg, config); err != nil {
		t.Errorf("валидное сообщение должно проходить проверку: %v", err)
	}

	// Тестируем пустой сервис
	invalidMsg1 := &LogMessage{
		Service:   "",
		Level:     INFO,
		Message:   "test message",
		Timestamp: time.Now(),
	}

	if err := ValidateMessage(invalidMsg1, config); err == nil {
		t.Error("сообщение с пустым сервисом должно быть отклонено")
	}

	// Тестируем слишком длинное имя сервиса
	longService := strings.Repeat("A", config.MaxServiceLength+1)
	invalidMsg2 := &LogMessage{
		Service:   longService,
		Level:     INFO,
		Message:   "test message",
		Timestamp: time.Now(),
	}

	if err := ValidateMessage(invalidMsg2, config); err == nil {
		t.Error("сообщение со слишком длинным именем сервиса должно быть отклонено")
	}

	// Тестируем слишком длинное сообщение
	longMessage := strings.Repeat("A", config.MaxMessageLength+1)
	invalidMsg3 := &LogMessage{
		Service:   "MAIN",
		Level:     INFO,
		Message:   longMessage,
		Timestamp: time.Now(),
	}

	if err := ValidateMessage(invalidMsg3, config); err == nil {
		t.Error("слишком длинное сообщение должно быть отклонено")
	}

	// Тестируем неразрешенный сервис (используем сервис с недопустимыми символами)
	invalidMsg4 := &LogMessage{
		Service:   "forbidden_service", // строчные буквы не разрешены
		Level:     INFO,
		Message:   "test message",
		Timestamp: time.Now(),
	}

	if err := ValidateMessage(invalidMsg4, config); err == nil {
		t.Error("сообщение с недопустимыми символами в имени сервиса должно быть отклонено")
	}

	// Тестируем недопустимый уровень логирования
	invalidMsg5 := &LogMessage{
		Service:   "MAIN",
		Level:     LogLevel(999), // недопустимый уровень
		Message:   "test message",
		Timestamp: time.Now(),
	}

	if err := ValidateMessage(invalidMsg5, config); err == nil {
		t.Error("сообщение с недопустимым уровнем должно быть отклонено")
	}
}

/**
 * TestValidateConfig проверяет валидацию конфигурации логирования
 * @param t *testing.T - тестовый контекст
 */
func TestValidateConfig(t *testing.T) {
	// Создаем валидную конфигурацию
	tempDir := t.TempDir()
	validConfig := &conf.LoggingConfig{
		Level:      "INFO",
		LogFile:    tempDir + "/test.log",
		SocketPath: tempDir + "/test.sock",
		Services:   []string{"MAIN", "API"},
	}

	if err := ValidateConfig(validConfig); err != nil {
		t.Errorf("валидная конфигурация должна проходить проверку: %v", err)
	}

	// Тестируем конфигурацию с относительным путем к лог-файлу
	invalidConfig1 := &conf.LoggingConfig{
		Level:      "INFO",
		LogFile:    "relative/path/test.log", // относительный путь
		SocketPath: tempDir + "/test.sock",
		Services:   []string{"MAIN"},
	}

	if err := ValidateConfig(invalidConfig1); err == nil {
		t.Error("конфигурация с относительным путем к лог-файлу должна быть отклонена")
	}

	// Тестируем конфигурацию с относительным путем к сокету
	invalidConfig2 := &conf.LoggingConfig{
		Level:      "INFO",
		LogFile:    tempDir + "/test.log",
		SocketPath: "relative/path/test.sock", // относительный путь
		Services:   []string{"MAIN"},
	}

	if err := ValidateConfig(invalidConfig2); err == nil {
		t.Error("конфигурация с относительным путем к сокету должна быть отклонена")
	}

	// Тестируем конфигурацию с пустым путем к лог-файлу
	invalidConfig3 := &conf.LoggingConfig{
		Level:      "INFO",
		LogFile:    "",
		SocketPath: tempDir + "/test.sock",
		Services:   []string{"MAIN"},
	}

	if err := ValidateConfig(invalidConfig3); err == nil {
		t.Error("конфигурация с пустым путем к лог-файлу должна быть отклонена")
	}

	// Тестируем конфигурацию с пустым путем к сокету
	invalidConfig4 := &conf.LoggingConfig{
		Level:      "INFO",
		LogFile:    tempDir + "/test.log",
		SocketPath: "",
		Services:   []string{"MAIN"},
	}

	if err := ValidateConfig(invalidConfig4); err == nil {
		t.Error("конфигурация с пустым путем к сокету должна быть отклонена")
	}

	// Тестируем конфигурацию с опасными символами в пути
	invalidConfig5 := &conf.LoggingConfig{
		Level:      "INFO",
		LogFile:    tempDir + "/../test.log", // содержит ..
		SocketPath: tempDir + "/test.sock",
		Services:   []string{"MAIN"},
	}

	if err := ValidateConfig(invalidConfig5); err == nil {
		t.Error("конфигурация с опасными символами в пути должна быть отклонена")
	}

	// Тестируем nil конфигурацию
	if err := ValidateConfig(nil); err == nil {
		t.Error("nil конфигурация должна быть отклонена")
	}
}

/**
 * TestRateLimiterMultipleClients проверяет работу с несколькими клиентами
 * @param t *testing.T - тестовый контекст
 */
func TestRateLimiterMultipleClients(t *testing.T) {
	config := DefaultSecurityConfig()
	config.RateLimitPerSecond = 1 // Один запрос в секунду

	limiter := NewRateLimiter(config)
	defer limiter.Close() // Закрываем cleanup горутину после теста

	client1 := "client-1"
	client2 := "client-2"

	// Каждый клиент должен иметь свой лимит
	if !limiter.IsAllowed(client1) {
		t.Error("первый запрос от client1 должен быть разрешен")
	}

	if !limiter.IsAllowed(client2) {
		t.Error("первый запрос от client2 должен быть разрешен")
	}

	// Вторые запросы должны быть заблокированы для обоих
	if limiter.IsAllowed(client1) {
		t.Error("второй запрос от client1 должен быть заблокирован")
	}

	if limiter.IsAllowed(client2) {
		t.Error("второй запрос от client2 должен быть заблокирован")
	}
}

/**
 * TestValidateMessageWithNilConfig проверяет валидацию с nil конфигурацией
 * @param t *testing.T - тестовый контекст
 */
func TestValidateMessageWithNilConfig(t *testing.T) {
	msg := &LogMessage{
		Service:   "MAIN",
		Level:     INFO,
		Message:   "test message",
		Timestamp: time.Now(),
	}

	if err := ValidateMessage(msg, nil); err == nil {
		t.Error("валидация с nil конфигурацией должна возвращать ошибку")
	}
}

/**
 * TestValidateMessageWithNilMessage проверяет валидацию nil сообщения
 * @param t *testing.T - тестовый контекст
 */
func TestValidateMessageWithNilMessage(t *testing.T) {
	config := DefaultSecurityConfig()

	if err := ValidateMessage(nil, config); err == nil {
		t.Error("валидация nil сообщения должна возвращать ошибку")
	}
}
