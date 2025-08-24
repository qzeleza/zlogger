// client_test.go - Unit тесты для LogClient
package logger

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	conf "kvasdns/internal/config"
)

// mockConn мок для net.Conn
type mockConn struct {
	readData  []byte
	writeBuf  []byte
	closed    bool
	readErr   error
	writeErr  error
	closeErr  error
	mu        sync.Mutex
	readIndex int
}

func newMockConn() *mockConn {
	return &mockConn{
		readData: []byte{},
		writeBuf: []byte{},
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.readErr != nil {
		return 0, m.readErr
	}

	if m.readIndex >= len(m.readData) {
		return 0, errors.New("конец данных")
	}

	n = copy(b, m.readData[m.readIndex:])
	m.readIndex += n
	return n, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.writeErr != nil {
		return 0, m.writeErr
	}

	m.writeBuf = append(m.writeBuf, b...)
	return len(b), nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return m.closeErr
}

func (m *mockConn) SetReadData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readData = data
	m.readIndex = 0
}

func (m *mockConn) GetWrittenData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.writeBuf
}

func (m *mockConn) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readErr = err
}

func (m *mockConn) SetWriteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writeErr = err
}

func (m *mockConn) SetCloseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closeErr = err
}

func (m *mockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closed
}

func (m *mockConn) LocalAddr() net.Addr                { return &net.UnixAddr{Name: "mock", Net: "unix"} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.UnixAddr{Name: "mock", Net: "unix"} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// TestNewLogClient проверяет создание нового клиента логгера
func TestNewLogClient(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Создаем мок соединения
	mockConn := newMockConn()

	// Подменяем функцию net.DialTimeout для возврата нашего мока
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return mockConn, nil
	}

	// Создаем конфигурацию
	config := &conf.LoggingConfig{
		SocketPath: "/tmp/logger.sock",
		Level:      "info",
	}

	// Создаем клиент
	client, err := NewLogClient(config)

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидалось успешное создание клиента, получена ошибка: %v", err)
	}

	if client == nil {
		t.Fatal("клиент не должен быть nil")
	}

	if client.config != config {
		t.Error("конфигурация должна быть установлена корректно")
	}

	if client.level != INFO {
		t.Errorf("ожидался уровень INFO, получен %v", client.level)
	}

	if client.conn == nil {
		t.Error("соединение должно быть установлено")
	}

	if client.encoder == nil {
		t.Error("энкодер должен быть создан")
	}

	if client.decoder == nil {
		t.Error("декодер должен быть создан")
	}

	if !client.connected {
		t.Error("флаг connected должен быть true")
	}

	if client.serviceLoggers == nil {
		t.Error("карта serviceLoggers должна быть инициализирована")
	}
}

// TestNewLogClientWithNilConfig проверяет создание клиента с nil конфигурацией
func TestNewLogClientWithNilConfig(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Создаем мок соединения
	mockConn := newMockConn()

	// Подменяем функцию net.DialTimeout для возврата нашего мока
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return mockConn, nil
	}

	// Создаем клиент с nil конфигурацией
	// Ожидаем ошибку, так как конфигурация по умолчанию может не содержать путь к сокету
	client, err := NewLogClient(nil)

	// Проверяем результаты - ожидаем ошибку из-за пустого пути к сокету
	if err == nil {
		t.Fatal("ожидалась ошибка при создании клиента с nil конфигурацией")
	}

	if client != nil {
		t.Fatal("клиент должен быть nil при ошибке создания")
	}
}

// TestNewLogClientConnectionError проверяет обработку ошибки подключения
func TestNewLogClientConnectionError(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Подменяем функцию net.DialTimeout для возврата ошибки
	expectedErr := errors.New("ошибка подключения")
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, expectedErr
	}

	// Создаем конфигурацию
	config := &conf.LoggingConfig{
		SocketPath: "/tmp/logger.sock",
		Level:      "info",
	}

	// Создаем клиент
	client, err := NewLogClient(config)

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}

	if client != nil {
		t.Error("клиент должен быть nil при ошибке подключения")
	}
}

// TestLogClientConnect проверяет метод connect
func TestLogClientConnect(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Создаем мок соединения
	mockConn := newMockConn()

	// Подменяем функцию net.DialTimeout для возврата нашего мока
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return mockConn, nil
	}

	// Создаем клиент без подключения
	client := &LogClient{
		config: &conf.LoggingConfig{
			SocketPath: "/tmp/logger.sock",
		},
		connected: false,
	}

	// Вызываем метод connect
	err := client.connect()

	// Проверяем результаты
	if err != nil {
		t.Fatalf("ожидалось успешное подключение, получена ошибка: %v", err)
	}

	if client.conn == nil {
		t.Error("соединение должно быть установлено")
	}

	if client.encoder == nil {
		t.Error("энкодер должен быть создан")
	}

	if client.decoder == nil {
		t.Error("декодер должен быть создан")
	}

	if !client.connected {
		t.Error("флаг connected должен быть true")
	}
}

// TestLogClientConnectError проверяет обработку ошибки в методе connect
func TestLogClientConnectError(t *testing.T) {
	// Сохраняем оригинальную функцию net.DialTimeout для восстановления после теста
	origDialTimeout := netDialTimeout
	defer func() { netDialTimeout = origDialTimeout }()

	// Подменяем функцию net.DialTimeout для возврата ошибки
	expectedErr := errors.New("ошибка подключения")
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, expectedErr
	}

	// Создаем клиент без подключения
	client := &LogClient{
		config: &conf.LoggingConfig{
			SocketPath: "/tmp/logger.sock",
		},
		connected: false,
	}

	// Вызываем метод connect
	err := client.connect()

	// Проверяем результаты
	if err == nil {
		t.Fatal("ожидалась ошибка, но получен nil")
	}

	if client.conn != nil {
		t.Error("соединение должно быть nil при ошибке")
	}

	if client.connected {
		t.Error("флаг connected должен быть false при ошибке")
	}
}
