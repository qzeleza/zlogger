// client_lifecycle_test.go - Тесты для методов жизненного цикла LogClient
package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	conf "kvasdns/internal/config"
)

// TestClientPing проверяет проверку соединения
func TestClientPing(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Готовим ответ сервера
	response := ProtocolMessage{
		Type: MsgTypePong,
		Data: "PONG",
	}

	responseData, _ := json.Marshal(response)
	mockConn.SetReadData(append(responseData, '\n'))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:      mockConn,
		encoder:   json.NewEncoder(mockConn),
		decoder:   json.NewDecoder(mockConn),
		connected: true,
	}

	// Вызываем метод Ping
	err := client.Ping()

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидался успешный пинг, получена ошибка: %v", err)
	}

	// Проверяем, что был отправлен правильный запрос
	writtenData := mockConn.GetWrittenData()
	var sentMsg ProtocolMessage
	decoder := json.NewDecoder(bytes.NewReader(writtenData))
	if err := decoder.Decode(&sentMsg); err != nil {
		t.Fatalf("ошибка декодирования отправленного запроса: %v", err)
	}

	if sentMsg.Type != MsgTypePing {
		t.Errorf("ожидался тип запроса %s, получен %s", MsgTypePing, sentMsg.Type)
	}

	sentDataStr, ok := sentMsg.Data.(string)
	if !ok {
		t.Fatal("данные запроса должны быть строкой")
	}

	if sentDataStr != "PING" {
		t.Errorf("ожидались данные запроса 'PING', получены '%s'", sentDataStr)
	}
}

// TestClientPingError проверяет обработку ошибки при пинге
func TestClientPingError(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Готовим ответ сервера с ошибкой
	response := ProtocolMessage{
		Type: MsgTypeError,
		Data: "ошибка пинга",
	}

	responseData, _ := json.Marshal(response)
	mockConn.SetReadData(append(responseData, '\n'))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:      mockConn,
		encoder:   json.NewEncoder(mockConn),
		decoder:   json.NewDecoder(mockConn),
		connected: true,
	}

	// Вызываем метод Ping
	err := client.Ping()

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}
}

// TestClientPingConnectionError проверяет обработку ошибки соединения при пинге
func TestClientPingConnectionError(t *testing.T) {
	// Создаем мок соединения с ошибкой записи
	mockConn := newMockConn()
	mockConn.SetWriteError(errors.New("ошибка записи"))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:      mockConn,
		encoder:   json.NewEncoder(mockConn),
		decoder:   json.NewDecoder(mockConn),
		connected: true,
	}

	// Вызываем метод Ping
	err := client.Ping()

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}

	// Проверяем, что клиент помечен как отключенный
	if client.connected {
		t.Error("флаг connected должен быть false после ошибки соединения")
	}
}

// TestClientClose проверяет закрытие соединения
func TestClientClose(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:      mockConn,
		connected: true,
	}

	// Вызываем метод Close
	_ = client.Close()

	// Проверяем, что соединение было закрыто
	if !mockConn.IsClosed() {
		t.Error("соединение должно быть закрыто")
	}

	// Проверяем, что клиент помечен как отключенный
	if client.connected {
		t.Error("флаг connected должен быть false после закрытия")
	}
}

// TestClientCloseNilConnection проверяет обработку nil-соединения при закрытии
func TestClientCloseNilConnection(t *testing.T) {
	// Создаем клиент с nil-соединением
	client := &LogClient{
		conn:      nil,
		connected: true,
	}

	// Вызываем метод Close - не должно быть паники
	_ = client.Close()

	// Проверяем, что клиент помечен как отключенный
	if client.connected {
		t.Error("флаг connected должен быть false после закрытия")
	}
}

// TestClientSetService проверяет установку сервиса для клиента
func TestClientSetService(t *testing.T) {
	// Создаем клиент с инициализацией карты serviceLoggers
	client := &LogClient{
		serviceLoggers: make(map[string]*ServiceLogger),
	}

	// Устанавливаем сервис
	serviceName := "TEST_SERVICE"
	logger := client.SetService(serviceName)

	// Проверяем, что возвращен ServiceLogger
	if logger == nil {
		t.Fatal("возвращенный логгер не должен быть nil")
	}

	// Проверяем, что сервис установлен правильно
	// logger уже имеет тип *ServiceLogger, приведение типа не требуется
	serviceLogger := logger
	if serviceLogger == nil {
		t.Fatal("возвращенный логгер не должен быть nil")
	}

	if serviceLogger.service != serviceName {
		t.Errorf("ожидался сервис '%s', получен '%s'", serviceName, serviceLogger.service)
	}

	if serviceLogger.client != client {
		t.Error("клиент в ServiceLogger должен быть тем же, что и исходный клиент")
	}
}

// TestReconnect проверяет переподключение к серверу
func TestReconnect(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Создаем мок соединения для переподключения
	mockConn := newMockConn()

	// Подменяем функцию net.DialTimeout
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return mockConn, nil
	}

	// Создаем клиент с конфигурацией
	client := &LogClient{
		config: &conf.LoggingConfig{
			SocketPath: "/tmp/logger.sock",
		},
		connected: false,
	}

	// Вызываем метод reconnect
	err := client.reconnect()

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидалось успешное переподключение, получена ошибка: %v", err)
	}

	// Проверяем, что соединение установлено
	if client.conn != mockConn {
		t.Error("соединение должно быть установлено")
	}

	// Проверяем, что клиент помечен как подключенный
	if !client.connected {
		t.Error("флаг connected должен быть true после успешного подключения")
	}

	// Проверяем, что encoder и decoder установлены
	if client.encoder == nil {
		t.Error("encoder должен быть установлен")
	}

	if client.decoder == nil {
		t.Error("decoder должен быть установлен")
	}
}

// TestReconnectError проверяет обработку ошибки переподключения
func TestReconnectError(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Подменяем функцию net.DialTimeout с ошибкой
	expectedError := errors.New("ошибка подключения")
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, expectedError
	}

	// Создаем клиент с конфигурацией
	client := &LogClient{
		config: &conf.LoggingConfig{
			SocketPath: "/tmp/logger.sock",
		},
		connected: false,
	}

	// Вызываем метод reconnect
	err := client.reconnect()

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}

	// Проверяем, что ошибка содержит информацию о неудачных попытках переподключения
	if !strings.Contains(err.Error(), "не удалось переподключиться после") {
		t.Errorf("ожидалась ошибка о неудачном переподключении, получена '%v'", err)
	}

	// Проверяем, что клиент помечен как отключенный
	if client.connected {
		t.Error("флаг connected должен быть false после ошибки подключения")
	}
}

// TestRecoverPanic проверяет обработку паники
func TestRecoverPanic(t *testing.T) {
	// Создаем клиент без конфигурации - это нормально для теста RecoverPanic,
	// так как мы модифицировали метод RecoverPanic для безопасной работы без полной инициализации
	client := &LogClient{
		level: DEBUG,
	}

	// Сохраняем оригинальный stderr и создаем буфер для перехвата
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Вызываем функцию, которая вызовет панику
	func() {
		// Устанавливаем отложенный вызов RecoverPanic
		defer client.RecoverPanic("TEST")

		// Вызываем панику
		panic("тестовая паника")
	}()

	// Закрываем pipe и восстанавливаем stderr
	_ = w.Close()
	os.Stderr = origStderr

	// Читаем перехваченный вывод
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	// Переменная не используется, но сохраняем для отладки
	_ = buf.String()

	// Проверяем, что паника была обработана и записана в stderr
	if !bytes.Contains(buf.Bytes(), []byte("тестовая паника")) {
		t.Error("сообщение о панике должно быть записано в stderr")
	}

	if !bytes.Contains(buf.Bytes(), []byte("TEST")) {
		t.Error("имя сервиса должно быть записано в stderr")
	}
}

// TestClientUpdateConfig проверяет обновление конфигурации
func TestClientUpdateConfig(t *testing.T) {
	// Создаем клиент с начальной конфигурацией
	initialConfig := &conf.LoggingConfig{
		SocketPath: "/tmp/old.sock",
		Level:      "INFO",
	}

	client := &LogClient{
		config: initialConfig,
		level:  INFO,
	}

	// Создаем новую конфигурацию
	newConfig := &conf.LoggingConfig{
		SocketPath: "/tmp/new.sock",
		Level:      "DEBUG",
	}

	// Вызываем метод UpdateConfig
	_ = client.UpdateConfig(newConfig)

	// Проверяем, что конфигурация была обновлена
	if client.config != newConfig {
		t.Error("конфигурация должна быть обновлена")
	}

	// Проверяем, что уровень логирования был обновлен
	if client.level != DEBUG {
		t.Errorf("ожидался уровень DEBUG, получен %v", client.level)
	}
}

// TestUpdateConfigNil проверяет обработку nil-конфигурации при обновлении
func TestUpdateConfigNil(t *testing.T) {
	// Создаем клиент с начальной конфигурацией
	initialConfig := &conf.LoggingConfig{
		SocketPath: "/tmp/old.sock",
		Level:      "INFO",
	}

	client := &LogClient{
		config: initialConfig,
		level:  INFO,
	}

	// Вызываем метод UpdateConfig с nil
	_ = client.UpdateConfig(nil)

	// Проверяем, что конфигурация не изменилась
	if client.config != initialConfig {
		t.Error("конфигурация не должна измениться при nil")
	}

	// Проверяем, что уровень логирования не изменился
	if client.level != INFO {
		t.Errorf("уровень не должен измениться, ожидался INFO, получен %v", client.level)
	}
}
