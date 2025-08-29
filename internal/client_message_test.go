// client_message_test.go - Тесты для методов отправки сообщений LogClient
package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// Константы для тестирования
// Используем константы из message.go
var (
	// Локальные константы только для тестов
	testMsgTypeAck = "ack"
	// testMsgTypeCmd  = "cmd"
	// testMsgTypeResp = "resp"
)

// TestSendMessage проверяет отправку сообщения логгера на сервер
func TestSendMessage(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Готовим ответ сервера
	response := ProtocolMessage{
		Type: testMsgTypeAck,
		Data: "OK",
	}

	responseData, _ := json.Marshal(response)
	mockConn.SetReadData(append(responseData, '\n'))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:           mockConn,
		encoder:        json.NewEncoder(mockConn),
		decoder:        json.NewDecoder(mockConn),
		level:          DEBUG, // Устанавливаем уровень DEBUG, чтобы все сообщения проходили
		connected:      true,
		config:         &LoggingConfig{SocketPath: "/tmp/test.sock"}, // Добавляем конфигурацию
		serviceLoggers: make(map[string]*ServiceLogger),              // Инициализируем карту сервисов
	}

	// Вызываем метод sendMessage
	service := "TEST"
	level := INFO
	message := "test message"

	err := client.sendMessage(service, level, message, nil)

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидалась успешная отправка, получена ошибка: %v", err)
	}

	// Проверяем, что было отправлено правильное сообщение
	writtenData := mockConn.GetWrittenData()
	var sentMsg ProtocolMessage
	decoder := json.NewDecoder(bytes.NewReader(writtenData))
	if err := decoder.Decode(&sentMsg); err != nil {
		t.Fatalf("ошибка декодирования отправленного сообщения: %v", err)
	}

	if sentMsg.Type != MsgTypeLog {
		t.Errorf("ожидался тип сообщения %s, получен %s", MsgTypeLog, sentMsg.Type)
	}

	// Проверяем содержимое сообщения
	logMsgData, err := json.Marshal(sentMsg.Data)
	if err != nil {
		t.Fatalf("ошибка маршалинга данных сообщения: %v", err)
	}

	var logMsg LogMessage
	if err := json.Unmarshal(logMsgData, &logMsg); err != nil {
		t.Fatalf("ошибка демаршалинга данных сообщения: %v", err)
	}

	if logMsg.Service != service {
		t.Errorf("ожидался сервис '%s', получен '%s'", service, logMsg.Service)
	}

	if logMsg.Level != level {
		t.Errorf("ожидался уровень %v, получен %v", level, logMsg.Level)
	}

	if logMsg.Message != message {
		t.Errorf("ожидалось сообщение '%s', получен '%s'", message, logMsg.Message)
	}
}

// TestSendMessageLevelFiltering проверяет фильтрацию по уровню логирования
func TestSendMessageLevelFiltering(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Создаем клиент с уровнем INFO
	client := &LogClient{
		conn:           mockConn,
		encoder:        json.NewEncoder(mockConn),
		decoder:        json.NewDecoder(mockConn),
		level:          INFO, // Устанавливаем уровень INFO
		connected:      true,
		config:         &LoggingConfig{SocketPath: "/tmp/test.sock"}, // Добавляем конфигурацию
		serviceLoggers: make(map[string]*ServiceLogger),              // Инициализируем карту сервисов
	}

	// Проверяем фильтрацию DEBUG сообщений
	err := client.sendMessage("TEST", DEBUG, "debug message", nil)

	// DEBUG сообщение должно быть отфильтровано
	if err != nil {
		t.Errorf("ожидался nil при фильтрации, получена ошибка: %v", err)
	}

	// Проверяем, что сообщение не было отправлено
	writtenData := mockConn.GetWrittenData()
	if len(writtenData) > 0 {
		t.Error("DEBUG сообщение не должно быть отправлено при уровне INFO")
	}

	// Проверяем, что INFO сообщение проходит
	mockConn.SetReadData([]byte(`{"Type":"ack","Data":"OK"}` + "\n"))

	err = client.sendMessage("TEST", INFO, "info message", nil)

	if err != nil {
		t.Errorf("ожидалась успешная отправка INFO сообщения, получена ошибка: %v", err)
	}

	// Проверяем, что сообщение было отправлено
	writtenData = mockConn.GetWrittenData()
	if len(writtenData) == 0 {
		t.Error("INFO сообщение должно быть отправлено при уровне INFO")
	}
}

// TestSendMessageConnectionError проверяет обработку ошибки соединения
func TestSendMessageConnectionError(t *testing.T) {
	// Создаем мок соединения с ошибкой записи
	mockConn := newMockConn()
	mockConn.SetWriteError(errors.New("ошибка записи"))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:           mockConn,
		encoder:        json.NewEncoder(mockConn),
		decoder:        json.NewDecoder(mockConn),
		level:          DEBUG,
		connected:      true,
		config:         &LoggingConfig{SocketPath: "/tmp/test.sock"}, // Добавляем конфигурацию
		serviceLoggers: make(map[string]*ServiceLogger),              // Инициализируем карту сервисов
	}

	// Сохраняем оригинальный stderr и создаем буфер для перехвата
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Вызываем метод sendMessage
	err := client.sendMessage("TEST", INFO, "test message", nil)

	// Закрываем pipe и восстанавливаем stderr
	_ = w.Close()
	os.Stderr = origStderr

	// Читаем перехваченный вывод
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	stderrOutput := buf.String()

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}

	// Проверяем, что сообщение было записано в stderr
	if !strings.Contains(stderrOutput, "test message") {
		t.Error("сообщение должно быть записано в stderr при ошибке соединения")
	}

	// Проверяем, что клиент помечен как отключенный
	if client.connected {
		t.Error("флаг connected должен быть false после ошибки соединения")
	}
}

// TestFallbackToStderr проверяет запись сообщения в stderr как резервный вариант
func TestFallbackToStderr(t *testing.T) {
	// Создаем клиент
	client := &LogClient{}

	// Сохраняем оригинальный stderr и создаем буфер для перехвата
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Вызываем метод fallbackToStderr
	service := "TEST"
	level := ERROR
	message := "error message"
	timestamp := time.Now()

	client.fallbackToStderr(service, level, message, timestamp, nil)

	// Закрываем pipe и восстанавливаем stderr
	_ = w.Close()
	os.Stderr = origStderr

	// Читаем перехваченный вывод
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	stderrOutput := buf.String()

	// Проверяем результаты
	if !strings.Contains(stderrOutput, service) {
		t.Errorf("вывод stderr должен содержать сервис '%s'", service)
	}

	if !strings.Contains(stderrOutput, level.String()) {
		t.Errorf("вывод stderr должен содержать уровень '%s'", level.String())
	}

	if !strings.Contains(stderrOutput, message) {
		t.Errorf("вывод stderr должен содержать сообщение '%s'", message)
	}
}

// TestSendMessageReconnect проверяет переподключение при ошибке соединения
func TestSendMessageReconnect(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Создаем первый мок соединения с ошибкой записи
	failedConn := newMockConn()
	failedConn.SetWriteError(errors.New("ошибка записи"))

	// Создаем второй мок соединения для переподключения
	successConn := newMockConn()
	successConn.SetReadData([]byte(`{"Type":"ack","Data":"OK"}` + "\n"))

	// Счетчик вызовов для переключения между соединениями
	callCount := 0
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		callCount++
		if callCount == 1 {
			return successConn, nil
		}
		return nil, errors.New("неожиданный вызов")
	}

	// Создаем клиент с первым моком соединения
	client := &LogClient{
		config: &LoggingConfig{
			SocketPath: "/tmp/logger.sock",
		},
		conn:           failedConn,
		encoder:        json.NewEncoder(failedConn),
		decoder:        json.NewDecoder(failedConn),
		level:          DEBUG,
		connected:      true,
		serviceLoggers: make(map[string]*ServiceLogger), // Инициализируем карту сервисов
	}

	// Вызываем метод sendMessage
	err := client.sendMessage("TEST", INFO, "test message", nil)

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидалась успешная отправка после переподключения, получена ошибка: %v", err)
	}

	// Проверяем, что было выполнено переподключение
	if callCount != 1 {
		t.Errorf("ожидался 1 вызов переподключения, получено %d", callCount)
	}

	// Проверяем, что соединение обновлено
	if client.conn != successConn {
		t.Error("соединение должно быть обновлено после переподключения")
	}

	// Проверяем, что клиент помечен как подключенный
	if !client.connected {
		t.Error("флаг connected должен быть true после успешного переподключения")
	}
}
