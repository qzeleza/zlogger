// internal/logger/api.go
package logger

type API interface {
	SetService(service string) *ServiceLogger
	
	// Универсальные методы логирования
	Debug(...interface{}) error
	Info(...interface{}) error
	Warn(...interface{}) error
	Error(...interface{}) error
	Fatal(...interface{}) error
	Panic(...interface{}) error
}
