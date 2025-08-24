// internal/logger/api.go
package logger

type API interface {
	SetService(service string) *ServiceLogger
	Debug(string, ...interface{}) error
	Error(string, ...interface{}) error
	Info(string, ...interface{}) error
	Warn(string, ...interface{}) error
	
	// Форматированные методы
	Debugf(format string, args ...interface{}) error
	Infof(format string, args ...interface{}) error
	Warnf(format string, args ...interface{}) error
	Errorf(format string, args ...interface{}) error
}
