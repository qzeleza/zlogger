// client_request_test.go - Тесты для методов запроса и уровней логирования LogClient
package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

// Константы теперь определены в message.go

// TestSendRequest проверяет отправку запроса серверу и ожидание ответа
func TestSendRequest(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Готовим ответ сервера
	expectedResponse := ProtocolMessage{
		Type: MsgTypePong,
		Data: "PONG",
	}

	responseData, _ := json.Marshal(expectedResponse)
	mockConn.SetReadData(append(responseData, '\n'))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:      mockConn,
		encoder:   json.NewEncoder(mockConn),
		decoder:   json.NewDecoder(mockConn),
		connected: true,
	}

	// Вызываем метод sendRequest
	msgType := MsgTypePing
	data := "PING"

	response, err := client.sendRequest(msgType, data)

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидалась успешная отправка запроса, получена ошибка: %v", err)
	}

	if response == nil {
		t.Fatal("ответ не должен быть nil")
	}

	if response.Type != expectedResponse.Type {
		t.Errorf("ожидался тип ответа %s, получен %s", expectedResponse.Type, response.Type)
	}

	// Проверяем, что был отправлен правильный запрос
	writtenData := mockConn.GetWrittenData()
	var sentMsg ProtocolMessage
	decoder := json.NewDecoder(bytes.NewReader(writtenData))
	if err := decoder.Decode(&sentMsg); err != nil {
		t.Fatalf("ошибка декодирования отправленного запроса: %v", err)
	}

	if sentMsg.Type != msgType {
		t.Errorf("ожидался тип запроса %s, получен %s", msgType, sentMsg.Type)
	}

	sentDataStr, ok := sentMsg.Data.(string)
	if !ok {
		t.Fatal("данные запроса должны быть строкой")
	}

	if sentDataStr != data {
		t.Errorf("ожидались данные запроса '%s', получены '%s'", data, sentDataStr)
	}
}

// TestSendRequestWriteError проверяет обработку ошибки записи
func TestSendRequestWriteError(t *testing.T) {
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

	// Вызываем метод sendRequest
	response, err := client.sendRequest(MsgTypePing, "PING")

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}

	if response != nil {
		t.Error("ответ должен быть nil при ошибке")
	}

	// Проверяем, что клиент помечен как отключенный
	if client.connected {
		t.Error("флаг connected должен быть false после ошибки записи")
	}
}

// TestSendRequestReadError проверяет обработку ошибки чтения
func TestSendRequestReadError(t *testing.T) {
	// Создаем мок соединения с ошибкой чтения
	mockConn := newMockConn()
	mockConn.SetReadError(errors.New("ошибка чтения"))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:      mockConn,
		encoder:   json.NewEncoder(mockConn),
		decoder:   json.NewDecoder(mockConn),
		connected: true,
	}

	// Вызываем метод sendRequest
	response, err := client.sendRequest(MsgTypePing, "PING")

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}

	if response != nil {
		t.Error("ответ должен быть nil при ошибке")
	}

	// Проверяем, что клиент помечен как отключенный
	if client.connected {
		t.Error("флаг connected должен быть false после ошибки чтения")
	}
}

// TestSetServerLevel проверяет установку уровня логирования на сервере
func TestSetServerLevel(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Готовим ответ сервера
	response := ProtocolMessage{
		Type: "ack",
		Data: "OK",
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

	// Вызываем метод SetServerLevel
	err := client.SetServerLevel(WARN)

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидалась успешная установка уровня, получена ошибка: %v", err)
	}

	// Проверяем, что был отправлен правильный запрос
	writtenData := mockConn.GetWrittenData()
	var sentMsg ProtocolMessage
	decoder := json.NewDecoder(bytes.NewReader(writtenData))
	if err := decoder.Decode(&sentMsg); err != nil {
		t.Fatalf("ошибка декодирования отправленного запроса: %v", err)
	}

	if sentMsg.Type != MsgTypeSetLevel {
		t.Errorf("ожидался тип запроса %s, получен %s", MsgTypeSetLevel, sentMsg.Type)
	}

	// Проверяем данные запроса
	sentDataStr, ok := sentMsg.Data.(string)
	if !ok {
		t.Fatal("данные запроса должны быть строкой")
	}

	if sentDataStr != WARN.String() {
		t.Errorf("ожидался уровень '%s', получен '%s'", WARN.String(), sentDataStr)
	}
}

// TestSetServerLevelError проверяет обработку ошибки при установке уровня на сервере
func TestSetServerLevelError(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Готовим ответ сервера с ошибкой
	response := ProtocolMessage{
		Type: MsgTypeError,
		Data: "ошибка установки уровня",
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

	// Вызываем метод SetServerLevel
	err := client.SetServerLevel(WARN)

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}
}

// TestClientGetLogFile проверяет получение пути к файлу лога
func TestClientGetLogFile(t *testing.T) {
	// Создаем мок соединения
	mockConn := newMockConn()

	// Готовим ответ сервера
	expectedPath := "/var/log/app.log"
	response := ProtocolMessage{
		Type: MsgTypeLogFile, // Используем правильный тип ответа
		Data: expectedPath,
	}

	responseData, _ := json.Marshal(response)
	mockConn.SetReadData(append(responseData, '\n'))

	// Создаем клиент с моком соединения и конфигурацией
	client := &LogClient{
		conn:           mockConn,
		encoder:        json.NewEncoder(mockConn),
		decoder:        json.NewDecoder(mockConn),
		connected:      true,
		config:         &LoggingConfig{},                // Добавляем пустую конфигурацию
		serviceLoggers: make(map[string]*ServiceLogger), // Инициализируем карту логгеров
	}

	// Вызываем метод GetLogFile
	path := client.GetLogFile()

	// Проверяем результаты
	if path != expectedPath {
		t.Errorf("ожидался путь '%s', получен '%s'", expectedPath, path)
	}

	// Проверяем, что был отправлен правильный запрос
	writtenData := mockConn.GetWrittenData()
	var sentMsg ProtocolMessage
	decoder := json.NewDecoder(bytes.NewReader(writtenData))
	if err := decoder.Decode(&sentMsg); err != nil {
		t.Fatalf("ошибка декодирования отправленного запроса: %v", err)
	}

	if sentMsg.Type != MsgTypeGetLogFile {
		t.Errorf("ожидался тип запроса %s, получен %s", MsgTypeGetLogFile, sentMsg.Type)
	}
}

// TestGetLogFileError проверяет обработку ошибки при получении пути к файлу лога
func TestGetLogFileError(t *testing.T) {
	// Создаем мок соединения с ошибкой
	mockConn := newMockConn()
	mockConn.SetWriteError(errors.New("ошибка записи"))

	// Создаем клиент с моком соединения
	client := &LogClient{
		conn:      mockConn,
		encoder:   json.NewEncoder(mockConn),
		decoder:   json.NewDecoder(mockConn),
		connected: true,
	}

	// Вызываем метод GetLogFile
	path := client.GetLogFile()

	// Проверяем результаты - должен вернуться пустой путь
	if path != "" {
		t.Errorf("ожидался пустой путь, получен '%s'", path)
	}
}
