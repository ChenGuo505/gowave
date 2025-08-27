package register

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdOption struct {
	Endpoints   []string
	DialTimeout time.Duration
	ServiceName string
	Host        string
	Port        int
}

type EtcdRegister struct {
	Client *clientv3.Client
}

func (r *EtcdRegister) CreateClient() error {
	option := &Option{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   option.Endpoints,
		DialTimeout: option.DialTimeout,
	})
	if err != nil {
		return err
	}
	r.Client = client
	return nil
}

func (r *EtcdRegister) RegisterService(service string, host string, port int) error {
	ctx, cancel := context.WithTimeout(r.Client.Ctx(), 3*time.Second)
	defer cancel()
	// key format: /services/service_name/instance_id
	instanceID := uuid.New().String()
	key := fmt.Sprintf("/services/%s/%s", service, instanceID)
	_, err := r.Client.Put(ctx, key, fmt.Sprintf("%s:%d", host, port))
	return err
}

func (r *EtcdRegister) GetInstance(service string) (string, error) {
	ctx, cancel := context.WithTimeout(r.Client.Ctx(), 3*time.Second)
	defer cancel()
	key := fmt.Sprintf("/services/%s/", service)
	resp, err := r.Client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return "", err
	}
	kvs := resp.Kvs
	if len(kvs) == 0 {
		return "", fmt.Errorf("service %s not found", service)
	}
	// simple load balancing: random selection
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	idx := rd.Intn(len(kvs))
	return string(kvs[idx].Value), nil
}

func (r *EtcdRegister) Close() error {
	return r.Client.Close()
}
