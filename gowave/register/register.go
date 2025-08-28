package register

import (
	"time"

	"github.com/ChenGuo505/gowave/config"
)

const (
	Nacos = "nacos"
	Etcd  = "etcd"
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
	switch config.RootConfig.RegisterCenter.Type {
	case Nacos:
		return &NacosRegister{}
	case Etcd:
		return &EtcdRegister{}
	default:
		return nil
	}
}
