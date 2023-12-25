package locker

import "log"

type Logger interface {
	Info(kvs ...interface{})
	Warn(kvs ...interface{})
}

type defaultLogger struct{}

func (inst *defaultLogger) Info(kvs ...any) {
	log.Default().Println(kvs...)
}

func (inst *defaultLogger) Warn(kvs ...any) {
	log.Default().Println(kvs...)
}
