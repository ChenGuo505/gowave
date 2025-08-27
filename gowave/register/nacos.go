package register

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type NacosRegister struct {
	Client naming_client.INamingClient
}

func (r *NacosRegister) CreateClient() error {
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(""), //当namespace是public时，此处填空字符串。
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig(
			"127.0.0.1",
			8848,
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		),
	}
	cli, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return err
	}
	r.Client = cli
	return nil
}

func (r *NacosRegister) RegisterService(service string, host string, port int) error {
	_, err := r.Client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: service,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "shanghai"},
	})
	return err
}

func (r *NacosRegister) GetInstance(service string) (string, error) {
	instance, err := r.Client.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: service,
	})
	if err != nil {
		return "", err
	}
	addr := fmt.Sprintf("%s:%d", instance.Ip, instance.Port)
	return addr, nil
}

func (r *NacosRegister) Close() error {
	return nil
}
