// server.go - Оптимизированный сервер логгера для embedded систем
package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	SERVER_LOGGER_NAME = "SLOG"
)

// LogServer серверная часть логгера с оптимизациями для embedded систем
type LogServer struct {
	// int64 поля и структуры с int64 в начале для правильного выравнивания на 32-битных архитектурах (MIPS)
	currentSize int64 // Текущий размер файла
	connCounter int64 // Счетчик подключений

	// Статистика работы (содержит int64 поля)
	stats ServerStats // Статистика сервера

	// Основная конфигурация
	config   *LoggingConfig
	file     *os.File
	listener net.Listener

	// Буферизация и производительность
	buffer     chan LogMessage // Буфер входящих сообщений
	writeBatch []LogMessage    // Пакет для пакетной записи
	batchMu    sync.Mutex      // Мьютекс для пакета

	// Управление жизненным циклом
	done    chan struct{}  // Канал для остановки
	stopped bool           // Флаг остановки сервера
	wg      sync.WaitGroup // Группа ожидания горутин

	// Синхронизация и потокобезопасность
	mu sync.RWMutex // Основной мьютекс

	// Метрики и мониторинг
	maxServiceLen int // Максимальная длина имени сервиса (для выравнивания)
	maxLevelLen   int // Максимальная длина уровня (для выравнивания)

	// Управление клиентами
	clients   map[net.Conn]string // Карта активных клиентов
	clientsMu sync.RWMutex        // Мьютекс для клиентов

	// Фильтрация и безопасность
	minLevel       LogLevel        // Минимальный уровень логирования
	rateLimiter    *RateLimiter    // Ограничитель скорости
	securityConfig *SecurityConfig // Конфигурация безопасности

	// Кеширование (новая функциональность)
	cache *LogCache // Кеш записей для быстрого доступа
}

// ServerStats статистика работы сервера
type ServerStats struct {
	// int64 поля в начале для правильного выравнивания на 32-битных архитектурах (MIPS)
	TotalMessages int64 // Общее количество обработанных сообщений
	TotalClients  int64 // Общее количество подключений
	MemoryUsage   int64 // Использование памяти в байтах
	FileRotations int64 // Количество ротаций файла
	CacheHits     int64 // Попадания в кеш
	CacheMisses   int64 // Промахи кеша

	// Остальные поля
	CurrentClients int32     // Текущее количество клиентов
	LastRotation   time.Time // Время последней ротации
	StartTime      time.Time // Время запуска сервера
}

// NewLogServer создает новый оптимизированный сервер логгера
// Использует упрощенную конфигурацию + фиксированные оптимальные значения
func NewLogServer(config *LoggingConfig) (*LogServer, error) {
	// Проверка на nil конфигурацию
	if config == nil {
		return nil, fmt.Errorf("конфигурация не может быть nil")
	}

	// Валидация упрощенной конфигурации
	if config.LogFile == "" {
		return nil, fmt.Errorf("не указан путь к файлу лога")
	}
	if config.SocketPath == "" {
		return nil, fmt.Errorf("не указан путь к сокету")
	}

	// Парсинг минимального уровня логирования
	minLevel, err := ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("невалидный уровень логирования '%s': %w", config.Level, err)
	}

	server := &LogServer{
		config:        config,
		buffer:        make(chan LogMessage, config.BufferSize),
		writeBatch:    make([]LogMessage, 0, DEFAULT_WRITE_BATCH_SIZE), // Константа
		done:          make(chan struct{}),
		maxServiceLen: 4, // минимум для "MAIN"
		maxLevelLen:   5, // минимум для "DEBUG"
		clients:       make(map[net.Conn]string),
		minLevel:      minLevel,

		// Используем фиксированные оптимальные значения вместо конфигурации
		rateLimiter:    NewRateLimiter(DefaultSecurityConfig()),
		securityConfig: DefaultSecurityConfig(),
		stats: ServerStats{
			StartTime: time.Now(),
		},
	}

	// Кеш всегда включен с оптимальными настройками для embedded
	server.cache = NewLogCache(DEFAULT_CACHE_SIZE, time.Duration(DEFAULT_CACHE_TTL)*time.Second)

	// Вычисляем максимальные длины названий сервисов для выравнивания
	// с целью симметричного отображения в логах
	for _, service := range config.Services {
		if len(service) > server.maxServiceLen {
			server.maxServiceLen = len(service)
		}
	}

	// Вычисляем максимальные длины названий уровней для выравнивания
	// с целью симметричного отображения в логах
	for _, level := range levelNames {
		if len(level) > server.maxLevelLen {
			server.maxLevelLen = len(level)
		}
	}

	// Инициализация файла лога
	if err := server.initLogFile(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации файла лога: %w", err)
	}

	// Инициализация сокета
	if err := server.initSocket(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации сокета: %w", err)
	}

	// Регистрируем финалайзер, чтобы гарантировать вызов Stop(),
	// даже если пользователь забудет явно остановить сервер.
	runtime.SetFinalizer(server, func(s *LogServer) {
		_ = s.Stop()
	})

	return server, nil
}

// initLogFile инициализирует файл лога с фиксированными правами доступа
func (s *LogServer) initLogFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Создаем директорию если не существует
	logDir := filepath.Dir(s.config.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории лога: %w", err)
	}

	// Открываем файл с фиксированными правами доступа (константа)
	file, err := os.OpenFile(s.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.FileMode(DEFAULT_FILE_PERMISSIONS))
	if err != nil {
		return fmt.Errorf("ошибка открытия файла лога: %w", err)
	}

	// Закрываем предыдущий файл если есть
	if s.file != nil {
		s.file.Close()
	}
	s.file = file

	// Получаем текущий размер файла
	if stat, err := file.Stat(); err == nil {
		s.currentSize = stat.Size()
	}

	return nil
}

// initSocket инициализирует unix socket с фиксированными правами доступа
func (s *LogServer) initSocket() error {
	// Удаляем существующий сокет
	os.Remove(s.config.SocketPath)

	// Создаем директорию для сокета
	socketDir := filepath.Dir(s.config.SocketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории сокета: %w", err)
	}

	// Создаем сокет
	listener, err := net.Listen("unix", s.config.SocketPath)
	if err != nil {
		return fmt.Errorf("ошибка создания unix сокета: %w", err)
	}

	// Устанавливаем фиксированные права доступа к сокету (константа)
	if err := os.Chmod(s.config.SocketPath, os.FileMode(DEFAULT_SOCKET_PERMISSIONS)); err != nil {
		listener.Close()
		return fmt.Errorf("ошибка установки прав доступа к сокету: %w", err)
	}

	s.listener = listener
	return nil
}

// Start запускает сервер логгера без вывода в консоль
func (s *LogServer) Start() error {
	// Инициализируем файл лога
	if err := s.initLogFile(); err != nil {
		return fmt.Errorf("ошибка инициализации файла лога: %w", err)
	}

	// Инициализируем сокет
	if err := s.initSocket(); err != nil {
		return fmt.Errorf("ошибка инициализации сокета: %w", err)
	}

	// Запускаем обработчик буфера с пакетной записью
	s.wg.Add(1)
	go s.optimizedBufferHandler()

	// Запускаем таймер сброса буфера
	s.wg.Add(1)
	go s.flushTimer()

	// Запускаем обработчик соединений
	// Запускаем мониторинг ресурсов с выводом статистики в лог
	go s.resourceMonitor()

	// Запускаем обработчик соединений
	s.wg.Add(1)
	go s.connectionHandler()

	// Логируем запуск сервера в лог файл
	startMsg := LogMessage{
		Service:   "SLOG",
		Level:     INFO,
		Message:   "Сервер логгера запущен",
		Timestamp: time.Now(),
		ClientID:  "server",
	}

	select {
	case s.buffer <- startMsg:
	default:
		// Если буфер полон, записываем напрямую
		s.writeMessage(startMsg)
	}

	return nil
}

// optimizedBufferHandler обработчик буфера с пакетной записью для производительности
func (s *LogServer) optimizedBufferHandler() {
	defer s.wg.Done()

	// Защита от нулевого интервала
	flushInterval := s.config.FlushInterval
	if flushInterval <= 0 {
		flushInterval = time.Second // Значение по умолчанию
	}

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case msg := <-s.buffer:
			s.batchMu.Lock()
			s.writeBatch = append(s.writeBatch, msg)

			// Записываем пакет если достигли оптимального размера или это критическое сообщение
			if len(s.writeBatch) >= DEFAULT_WRITE_BATCH_SIZE || msg.Level >= ERROR {
				s.flushBatch()
			}
			s.batchMu.Unlock()

		case <-ticker.C:
			// Периодически сбрасываем пакет
			s.batchMu.Lock()
			if len(s.writeBatch) > 0 {
				s.flushBatch()
			}
			s.batchMu.Unlock()

		case <-s.done:
			// Записываем оставшиеся сообщения при остановке
			s.batchMu.Lock()
			if len(s.writeBatch) > 0 {
				s.flushBatch()
			}

			// Обрабатываем оставшиеся сообщения в буфере
			for len(s.buffer) > 0 {
				msg := <-s.buffer
				s.writeBatch = append(s.writeBatch, msg)
				if len(s.writeBatch) >= DEFAULT_WRITE_BATCH_SIZE {
					s.flushBatch()
				}
			}

			if len(s.writeBatch) > 0 {
				s.flushBatch()
			}
			s.batchMu.Unlock()
			return
		}
	}
}

// flushBatch записывает пакет сообщений на диск в TXT формате
// Использует константу DEFAULT_WRITE_BATCH_SIZE вместо конфигурации
func (s *LogServer) flushBatch() {
	if len(s.writeBatch) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		s.writeBatch = s.writeBatch[:0] // Очищаем пакет
		return
	}

	// Создаем буфер для пакетной записи в TXT формате
	var builder strings.Builder
	builder.Grow(len(s.writeBatch) * 100) // Примерная оценка размера

	for _, msg := range s.writeBatch {
		// ВАЖНО: Здесь используется TXT формат для записи в лог файл!
		formattedMsg := s.formatMessageAsTXT(msg)
		builder.WriteString(formattedMsg)
		builder.WriteString("\n")

		// Добавляем в кеш (кеш всегда включен с оптимальными настройками)
		if s.cache != nil {
			entry := LogEntry{
				Service:   msg.Service,
				Level:     msg.Level,
				Message:   msg.Message,
				Timestamp: msg.Timestamp,
				Raw:       formattedMsg,
			}
			cacheKey := fmt.Sprintf("%s_%d", msg.Service, msg.Timestamp.Unix())
			s.cache.Put(cacheKey, entry)
		}

		// Освобождаем объект сообщения в пул
		PutLogMessage(&msg)
	}

	// Записываем весь пакет одним вызовом в TXT формате
	data := builder.String()
	n, err := s.file.WriteString(data)
	if err != nil {
		// Логируем ошибку в stderr как fallback
		fmt.Fprintf(os.Stderr, "Ошибка записи в лог: %v\n", err)
	} else {
		s.currentSize += int64(n)
		atomic.AddInt64(&s.stats.TotalMessages, int64(len(s.writeBatch)))
	}

	// Очищаем пакет для переиспользования
	s.writeBatch = s.writeBatch[:0]

	// Проверяем необходимость ротации (MaxFileSize в мегабайтах)
	maxSizeBytes := int64(s.config.MaxFileSize * 1024 * 1024)
	if s.currentSize >= maxSizeBytes {
		_ = s.rotateIfNeeded()
	}
}

// formatMessageAsTXT форматирует сообщение в простой TXT формат для файла лога
// Формат: [SERVICE] YYYY-MM-DD HH:MM:SS [LEVEL] "MESSAGE"
// Если есть дополнительные поля, они выводятся с отступом на новых строках
func (s *LogServer) formatMessageAsTXT(msg LogMessage) string {
	service := fmt.Sprintf("%-*s", s.maxServiceLen, msg.Service)
	level := fmt.Sprintf("%-*s", s.maxLevelLen, msg.Level.String())
	timeStr := msg.Timestamp.Format(DEFAULT_TIME_FORMAT) // Фиксированный формат времени

	result := fmt.Sprintf("[%s] %s [%s] \"%s\"", service, timeStr, level, msg.Message)
	
	// Если есть дополнительные поля, добавляем их с отступом
	if len(msg.Fields) > 0 {
		keys := make([]string, 0, len(msg.Fields))
		for k := range msg.Fields {
			keys = append(keys, k)
		}
		
		// Сортируем ключи для стабильного вывода
		sort.Strings(keys)
		
		for _, k := range keys {
			result += fmt.Sprintf("\n    %s: %s", k, msg.Fields[k])
		}
	}
	
	return result
}

// connectionHandler обрабатывает входящие соединения с защитой от DoS
func (s *LogServer) connectionHandler() {
	defer s.wg.Done()

	for {
		select {
		case <-s.done:
			return
		default:
			// Устанавливаем таймаут на прием соединения
			if tcpListener, ok := s.listener.(*net.TCPListener); ok {
				_ = tcpListener.SetDeadline(time.Now().Add(time.Second))
			}

			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.done:
					return
				default:
					// Игнорируем таймауты и продолжаем
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue
					}
					time.Sleep(100 * time.Millisecond) // Небольшая задержка при ошибке
					continue
				}
			}

			// Проверяем лимит подключений (используем константу)
			s.clientsMu.RLock()
			clientCount := len(s.clients)
			s.clientsMu.RUnlock()

			if clientCount >= DEFAULT_MAX_CONNECTIONS {
				conn.Close()
				continue
			}

			// Регистрируем клиента
			clientID := fmt.Sprintf("client_%d", atomic.AddInt64(&s.connCounter, 1))
			s.clientsMu.Lock()
			s.clients[conn] = clientID
			s.clientsMu.Unlock()

			atomic.AddInt32(&s.stats.CurrentClients, 1)
			atomic.AddInt64(&s.stats.TotalClients, 1)

			go s.handleClient(conn, clientID)
		}
	}
}

// handleClient обрабатывает отдельного клиента с защитой от атак
func (s *LogServer) handleClient(conn net.Conn, clientID string) {
	defer func() {
		conn.Close()
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()

		atomic.AddInt32(&s.stats.CurrentClients, -1)
	}()

	// Устанавливаем фиксированные таймауты для защиты от hanging connections
	timeout := time.Duration(DEFAULT_CONNECTION_TIMEOUT) * time.Second
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))

	encoder := json.NewEncoder(conn)
	// Ограничиваем размер входящих данных (константа)
	decoder := json.NewDecoder(&io.LimitedReader{
		R: conn,
		N: int64(DEFAULT_MAX_MESSAGE_SIZE),
	})

	for {
		select {
		case <-s.done:
			return
		default:
			// Проверяем rate limiting
			if !s.rateLimiter.IsAllowed(clientID) {
				s.sendError(encoder, "Превышен лимит скорости сообщений")
				time.Sleep(time.Second) // Замедляем спамера
				continue
			}

			// Обновляем таймаут чтения (константа)
			timeout := time.Duration(DEFAULT_CONNECTION_TIMEOUT) * time.Second
			_ = conn.SetReadDeadline(time.Now().Add(timeout))

			var protocolMsg ProtocolMessage
			if err := decoder.Decode(&protocolMsg); err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					s.sendError(encoder, "Таймаут чтения")
				}
				return
			}

			// Обрабатываем сообщение в зависимости от типа
			switch protocolMsg.Type {
			case MsgTypeLog:
				s.handleLogMessage(protocolMsg.Data, clientID)

			case MsgTypeGetEntries:
				s.handleGetEntries(protocolMsg.Data, encoder)

			case MsgTypeUpdateLevel:
				s.handleUpdateLevel(protocolMsg.Data, encoder)

			case MsgTypeSetLevel:
				// Обрабатываем так же, как и MsgTypeUpdateLevel, так как они выполняют одинаковую функцию
				s.handleUpdateLevel(protocolMsg.Data, encoder)

			case MsgTypePing:
				s.handlePing(encoder)

			case MsgTypeGetLogFile:
				// Обработка запроса на получение пути к файлу лога
				response := ProtocolMessage{
					Type: MsgTypeLogFile,
					Data: s.config.LogFile,
				}
				_ = encoder.Encode(response)

			default:
				s.sendError(encoder, fmt.Sprintf("Неизвестный тип сообщения: %s", protocolMsg.Type))
			}
		}
	}
}

// handleLogMessage обрабатывает сообщение лога с валидацией
func (s *LogServer) handleLogMessage(data interface{}, clientID string) {
	// Получаем объект сообщения из пула
	msg := GetLogMessage()
	defer PutLogMessage(msg)

	// Сериализуем и десериализуем данные
	msgData, err := json.Marshal(data)
	if err != nil {
		return
	}

	if err := json.Unmarshal(msgData, msg); err != nil {
		return
	}

	// Валидация сообщения (без вывода в консоль)
	if err := ValidateMessage(msg, s.securityConfig); err != nil {
		return
	}

	// Проверяем уровень логирования
	if msg.Level < s.minLevel {
		return
	}

	// Проверяем ограничения на сервисы (без вывода в консоль)
	if s.config.RestrictServices {
		allowed := false
		for _, service := range s.config.Services {
			if msg.Service == service {
				allowed = true
				break
			}
		}
		if !allowed {
			return
		}
	}

	msg.ClientID = clientID
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Отправляем в буфер (неблокирующая отправка)
	select {
	case s.buffer <- *msg:
	default:
		// Буфер переполнен - пропускаем сообщение или записываем напрямую для критических
		if msg.Level >= ERROR {
			s.writeMessage(*msg)
		}
	}
}

// sendError отправляет ошибку клиенту
func (s *LogServer) sendError(encoder *json.Encoder, message string) {
	response := ProtocolMessage{
		Type: MsgTypeError,
		Data: message,
	}
	_ = encoder.Encode(response)
}

// handleGetEntries обрабатывает запрос на получение записей лога
func (s *LogServer) handleGetEntries(data interface{}, encoder *json.Encoder) {
	filterData, err := json.Marshal(data)
	if err != nil {
		s.sendError(encoder, "Неверные данные фильтра")
		return
	}

	var filter FilterOptions
	if err := json.Unmarshal(filterData, &filter); err != nil {
		s.sendError(encoder, "Неверный формат фильтра")
		return
	}

	// Валидация фильтра
	if err := filter.Validate(); err != nil {
		s.sendError(encoder, fmt.Sprintf("Ошибка валидации фильтра: %v", err))
		return
	}

	entries, err := s.getLogEntries(filter)
	if err != nil {
		s.sendError(encoder, fmt.Sprintf("Ошибка получения записей: %v", err))
		return
	}

	response := ProtocolMessage{
		Type: MsgTypeResponse,
		Data: entries,
	}
	_ = encoder.Encode(response)
}

// handleUpdateLevel обрабатывает обновление уровня логирования
func (s *LogServer) handleUpdateLevel(data interface{}, encoder *json.Encoder) {
	levelData, err := json.Marshal(data)
	if err != nil {
		s.sendError(encoder, "Неверные данные уровня")
		return
	}

	var levelStr string
	if err := json.Unmarshal(levelData, &levelStr); err != nil {
		s.sendError(encoder, "Неверный формат уровня")
		return
	}

	level, err := ParseLevel(levelStr)
	if err != nil {
		s.sendError(encoder, fmt.Sprintf("Недопустимый уровень: %v", err))
		return
	}

	// Обновляем минимальный уровень логирования
	s.mu.Lock()
	s.minLevel = level
	s.mu.Unlock()

	// Логируем изменение уровня
	changeMsg := LogMessage{
		Service:   SERVER_LOGGER_NAME,
		Level:     INFO,
		Message:   fmt.Sprintf("Уровень логирования изменен на %s", level.String()),
		Timestamp: time.Now(),
		ClientID:  "server",
	}

	select {
	case s.buffer <- changeMsg:
	default:
		s.writeMessage(changeMsg)
	}

	response := ProtocolMessage{
		Type: MsgTypeResponse,
		Data: "Уровень логирования обновлен",
	}
	_ = encoder.Encode(response)
}

// flushTimer периодически сбрасывает буфер на диск для надежности
func (s *LogServer) flushTimer() {
	defer s.wg.Done()

	// Защита от нулевого интервала
	flushInterval := s.config.FlushInterval
	if flushInterval <= 0 {
		flushInterval = time.Second // Значение по умолчанию
	}

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Принудительно сбрасываем накопленные данные
			s.flush()
		case <-s.done:
			// Финальный сброс при остановке
			s.flush()
			return
		}
	}
}

// flush сбрасывает буфер на диск
func (s *LogServer) flush() {
	// Сначала сбрасываем пакет сообщений из writeBatch
	s.batchMu.Lock()
	if len(s.writeBatch) > 0 {
		s.flushBatch()
	}
	s.batchMu.Unlock()

	// Затем синхронизируем файл
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		_ = s.file.Sync()
	}
}

// handlePing обрабатывает ping запрос для проверки соединения
func (s *LogServer) handlePing(encoder *json.Encoder) {
	response := ProtocolMessage{
		Type: MsgTypePong,
		Data: "pong",
	}
	_ = encoder.Encode(response)
}

// Flush публичный метод для принудительного сброса буфера
func (s *LogServer) Flush() {
	s.flush()
}

// Stop останавливает сервер логгера
func (s *LogServer) Stop() error {
	// Первая критическая секция: помечаем остановку и копируем необходимые указатели,
	// чтобы дальнейшие действия выполнять без удержания глобального мьютекса.
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true

	// Формируем сообщение об остановке (без затратных операций внутри локов).
	stopMsg := LogMessage{
		Service:   SERVER_LOGGER_NAME,
		Level:     INFO,
		Message:   "Сервер логгера останавливается",
		Timestamp: time.Now(),
		ClientID:  "server",
	}

	// Сохраняем ссылки для дальнейшего корректного завершения.
	listener := s.listener
	s.listener = nil
	file := s.file
	s.file = nil

	// Закрываем done-канал один раз.
	close(s.done)
	s.mu.Unlock()

	// Отправляем сообщение об остановке без риска блокировок.
	select {
	case s.buffer <- stopMsg:
	default:
		// Если буфер переполнен, выполняем прямую запись.
		s.writeMessage(stopMsg)
	}

	// Закрываем сетевой слушатель.
	if listener != nil {
		_ = listener.Close()
	}

	// Закрываем все клиентские соединения.
	s.clientsMu.Lock()
	for conn := range s.clients {
		_ = conn.Close()
	}
	s.clientsMu.Unlock()

	// Останавливаем вспомогательные подсистемы (RateLimiter, LogCache).
	if s.rateLimiter != nil {
		s.rateLimiter.Close()
	}
	if s.cache != nil {
		s.cache.Close()
	}

	// Дожидаемся завершения всех горутин сервера.
	s.wg.Wait()

	// Закрываем файл лога после полного завершения горутин, чтобы избежать ошибок "file already closed".
	if file != nil {
		_ = file.Close()
	}

	// Удаляем сокетный файл.
	_ = os.Remove(s.config.SocketPath)

	return nil
}

// getLogEntries читает записи из лога с фильтрацией
func (s *LogServer) getLogEntries(filter FilterOptions) ([]LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, err := os.Open(s.config.LogFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла лога: %w", err)
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		entry, err := s.parseLogEntry(line)
		if err != nil {
			continue // пропускаем некорректные строки
		}

		// Применяем фильтры
		if !s.matchesFilter(entry, filter) {
			continue
		}

		entries = append(entries, entry)

		// Применяем лимит
		if filter.Limit > 0 && len(entries) >= filter.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения файла лога: %w", err)
	}

	return entries, nil
}

// parseLogEntry парсит строку лога в LogEntry
func (s *LogServer) parseLogEntry(line string) (LogEntry, error) {
	// Ожидаемый формат: [SERVICE] YYYY-MM-DD HH:MM:SS [LEVEL] "MESSAGE"

	if len(line) < 10 {
		return LogEntry{}, fmt.Errorf("строка слишком короткая")
	}

	// Находим первую закрывающую скобку для сервиса
	serviceEnd := strings.Index(line, "]")
	if serviceEnd == -1 || serviceEnd < 2 {
		return LogEntry{}, fmt.Errorf("неверный формат сервиса")
	}

	service := strings.TrimSpace(line[1:serviceEnd])

	// Пропускаем пробел после сервиса
	remaining := strings.TrimSpace(line[serviceEnd+1:])

	// Находим начало уровня
	levelStart := strings.Index(remaining, "[")
	if levelStart == -1 {
		return LogEntry{}, fmt.Errorf("уровень не найден")
	}

	// Извлекаем время (используем фиксированный формат)
	timeStr := strings.TrimSpace(remaining[:levelStart])
	timestamp, err := time.Parse(DEFAULT_TIME_FORMAT, timeStr)
	if err != nil {
		return LogEntry{}, fmt.Errorf("неверный формат времени: %w", err)
	}

	// Находим конец уровня
	levelEnd := strings.Index(remaining[levelStart:], "]")
	if levelEnd == -1 {
		return LogEntry{}, fmt.Errorf("неверный формат уровня")
	}
	levelEnd += levelStart

	levelStr := strings.TrimSpace(remaining[levelStart+1 : levelEnd])
	level, err := ParseLevel(levelStr)
	if err != nil {
		return LogEntry{}, fmt.Errorf("недопустимый уровень: %w", err)
	}

	// Извлекаем сообщение
	messageStart := strings.Index(remaining[levelEnd:], "\"")
	if messageStart == -1 {
		return LogEntry{}, fmt.Errorf("сообщение не найдено")
	}
	messageStart += levelEnd

	messageEnd := strings.LastIndex(remaining, "\"")
	if messageEnd == -1 || messageEnd <= messageStart {
		return LogEntry{}, fmt.Errorf("неверный формат сообщения")
	}

	message := remaining[messageStart+1 : messageEnd]

	return LogEntry{
		Service:   service,
		Level:     level,
		Message:   message,
		Timestamp: timestamp,
		Raw:       line,
	}, nil
}

// matchesFilter проверяет соответствие записи фильтру
func (s *LogServer) matchesFilter(entry LogEntry, filter FilterOptions) bool {
	// Фильтр по времени
	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}

	// Фильтр по уровню
	if filter.Level != nil && entry.Level != *filter.Level {
		return false
	}

	// Фильтр по сервису
	if filter.Service != "" && entry.Service != filter.Service {
		return false
	}

	return true
}

// resourceMonitor мониторит использование ресурсов и записывает статистику в лог
func (s *LogServer) resourceMonitor() {
	// Регистрируемся в WaitGroup, чтобы Stop() корректно дожидался завершения
	s.wg.Add(1)

	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute) // Проверяем каждую минуту
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.GC() // Принудительная сборка мусора для точных измерений
			runtime.ReadMemStats(&memStats)

			atomic.StoreInt64(&s.stats.MemoryUsage, int64(memStats.Alloc))

			// Проверяем лимит памяти
			if int64(memStats.Alloc) > DEFAULT_MAX_MEMORY {
				// Принимаем меры: очищаем кеш, собираем мусор
				if s.cache != nil {
					s.cache.Clear()
				}
				runtime.GC()
			}

			// Записываем статистику в лог каждые 10 минут в JSON формате
			if time.Now().Minute()%10 == 0 {
				s.logStatsAsJSON()
			}

		case <-s.done:
			return
		}
	}
}

// logStatsAsJSON записывает статистику в лог файл в JSON формате
func (s *LogServer) logStatsAsJSON() {
	uptime := time.Since(s.stats.StartTime)

	// Формируем JSON статистику
	statsData := map[string]interface{}{
		"type":            "server_stats",
		"uptime_seconds":  int(uptime.Seconds()),
		"total_messages":  atomic.LoadInt64(&s.stats.TotalMessages),
		"total_clients":   atomic.LoadInt64(&s.stats.TotalClients),
		"current_clients": atomic.LoadInt32(&s.stats.CurrentClients),
		"memory_usage_mb": float64(atomic.LoadInt64(&s.stats.MemoryUsage)) / 1024 / 1024,
		"file_rotations":  atomic.LoadInt64(&s.stats.FileRotations),
		"timestamp":       time.Now().Format(DEFAULT_TIME_FORMAT),
	}

	// Добавляем статистику кеша если есть
	if s.cache != nil {
		cacheStats := s.cache.GetStats()
		hitRate := float64(0)
		if cacheStats.Hits+cacheStats.Misses > 0 {
			hitRate = float64(cacheStats.Hits) / float64(cacheStats.Hits+cacheStats.Misses) * 100
		}
		statsData["cache_size"] = cacheStats.Size
		statsData["cache_hit_rate"] = hitRate
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(statsData)
	if err != nil {
		return
	}

	// Записываем как специальное сообщение статистики
	statsMsg := LogMessage{
		Service:   SERVER_LOGGER_NAME,
		Level:     INFO,
		Message:   string(jsonData),
		Timestamp: time.Now(),
		ClientID:  "server",
	}

	select {
	case s.buffer <- statsMsg:
	default:
		// Если буфер полон, записываем напрямую
		s.writeMessage(statsMsg)
	}
}

// writeMessage записывает одно сообщение напрямую (для критических ситуаций)
func (s *LogServer) writeMessage(msg LogMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return
	}

	formattedMsg := s.formatMessageAsTXT(msg)
	n, err := s.file.WriteString(formattedMsg + "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка записи в лог: %v\n", err)
		return
	}

	s.currentSize += int64(n)
	atomic.AddInt64(&s.stats.TotalMessages, 1)

	// Принудительная синхронизация для критических сообщений
	if msg.Level >= ERROR {
		_ = s.file.Sync()
	}

	// Проверяем необходимость ротации (MaxFileSize в мегабайтах)
	maxSizeBytes := int64(s.config.MaxFileSize * 1024 * 1024)
	if s.currentSize >= maxSizeBytes {
		_ = s.rotateIfNeeded()
	}
}

// rotateIfNeeded выполняет ротацию логов при необходимости
func (s *LogServer) rotateIfNeeded() error {
	atomic.AddInt64(&s.stats.FileRotations, 1)
	s.stats.LastRotation = time.Now()

	if s.config.MaxFiles <= 1 {
		// Просто очищаем файл
		if s.file != nil {
			s.file.Close()
		}

		file, err := os.OpenFile(s.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(DEFAULT_FILE_PERMISSIONS))
		if err != nil {
			return err
		}

		s.file = file
		s.currentSize = 0
		return nil
	}

	// Закрываем текущий файл
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}

	// Перемещаем файлы (ротация)
	for i := s.config.MaxFiles - 2; i >= 0; i-- {
		oldName := s.config.LogFile
		if i > 0 {
			oldName = fmt.Sprintf("%s.%d", s.config.LogFile, i)
		}

		newName := fmt.Sprintf("%s.%d", s.config.LogFile, i+1)

		if _, err := os.Stat(oldName); err == nil {
			_ = os.Rename(oldName, newName)
		}
	}

	// Создаем новый файл
	file, err := os.OpenFile(s.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(DEFAULT_FILE_PERMISSIONS))
	if err != nil {
		return err
	}

	s.file = file
	s.currentSize = 0

	return nil
}
