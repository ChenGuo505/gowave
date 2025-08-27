package register

import "time"

type Option struct {
	Endpoints   []string
	DialTimeout time.Duration
	ServiceName string
	Host        string
	Port        int
}

type Register interface {
	CreateClient() error
	RegisterService(service string, host string, port int) error
	GetInstance(service string) (string, error)
	Close() error
}
