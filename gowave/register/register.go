package register

import (
	"time"

	"github.com/ChenGuo505/gowave/config"
)

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

func LoadRegister() Register {
	if config.RootConfig.Etcd != nil {
		return &EtcdRegister{}
	}
	if config.RootConfig.Nacos != nil {
		return &NacosRegister{}
	}
	return nil
}
