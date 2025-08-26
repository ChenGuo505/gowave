package rpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type GrpcServer struct {
	listener net.Listener
	server   *grpc.Server
	register []func(server *grpc.Server)
	opts     []grpc.ServerOption
}

func NewGrpcServer(addr string, opts ...GrpcOption) (*GrpcServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server := grpc.NewServer()
	gs := &GrpcServer{
		listener: listener,
		server:   server,
	}
	for _, opt := range opts {
		opt.Apply(gs)
	}
	return gs, nil
}

func (gs *GrpcServer) Serve() error {
	for _, r := range gs.register {
		r(gs.server)
	}
	return gs.server.Serve(gs.listener)
}

func (gs *GrpcServer) Stop() {
	gs.server.Stop()
}

func (gs *GrpcServer) Register(r func(server *grpc.Server)) {
	gs.register = append(gs.register, r)
}

type GrpcOption interface {
	Apply(*GrpcServer)
}

type DefaultGrpcOption struct {
	f func(*GrpcServer)
}

func (d *DefaultGrpcOption) Apply(gs *GrpcServer) {
	d.f(gs)
}

func WithOptions(opts ...grpc.ServerOption) GrpcOption {
	return &DefaultGrpcOption{
		f: func(gs *GrpcServer) {
			gs.opts = append(gs.opts, opts...)
		},
	}
}

type GrpcClient struct {
	conn *grpc.ClientConn
}

func NewGrpcClient(conf *GrpcClientConfig) (*GrpcClient, error) {
	ctx := context.Background()
	dialOptions := conf.dialOptions

	if conf.DialTimeout > time.Duration(0) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, conf.DialTimeout)
		defer cancel()
	}
	if conf.KeepAlive != nil {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(*conf.KeepAlive))
	}
	conn, err := grpc.NewClient(conf.Addr, dialOptions...)
	if err != nil {
		return nil, err
	}
	return &GrpcClient{
		conn: conn,
	}, nil
}

type GrpcClientConfig struct {
	Addr        string
	DialTimeout time.Duration
	ReadTimeout time.Duration
	IsDirect    bool
	KeepAlive   *keepalive.ClientParameters
	dialOptions []grpc.DialOption
}

func DefaultGrpcClientConfig() *GrpcClientConfig {
	return &GrpcClientConfig{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		DialTimeout: 10 * time.Second,
		ReadTimeout: 10 * time.Second,
	}
}
