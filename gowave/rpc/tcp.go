package rpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"

	"github.com/ChenGuo505/gowave/log"
	"github.com/ChenGuo505/gowave/register"
	"golang.org/x/time/rate"
)

type SerializerProtocol byte
type CompressorProtocol byte
type MessageType byte

const (
	SerializerGob SerializerProtocol = iota
)

const (
	CompressorGzip CompressorProtocol = iota
)

const (
	msgRequest = iota
	msgResponse
)

const MagicNumber = 0x1
const Version = 0x1

type Serializer interface {
	Serialize(v any) ([]byte, error)
	Deserialize(data []byte, v any) error
}

type GobSerializer struct{}

func (g GobSerializer) Serialize(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g GobSerializer) Deserialize(data []byte, v any) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(v)
}

type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}

type GzipCompressor struct{}

func (g GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g GzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	defer func(reader *gzip.Reader) {
		err := reader.Close()
		if err != nil {
			panic(err)
		}
	}(reader)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Header struct {
	MagicNumber    byte
	Version        byte
	FullLength     int32
	MsgType        MessageType
	CompressorType CompressorProtocol
	SerializerType SerializerProtocol
	RequestID      int64
}

type Message struct {
	Header *Header
	Data   any
}

type Request struct {
	RequestID   int64
	ServiceName string
	MethodName  string
	Args        []any
}

type Response struct {
	RequestID      int64
	Code           int16
	Msg            string
	CompressorType CompressorProtocol
	SerializerType SerializerProtocol
	Data           any
}

type GWRpcServer interface {
	Register(name string, service any)
	Start()
	Stop()
}

type RegisterCenter string

type TcpServer struct {
	host       string
	port       uint64
	listener   net.Listener
	serviceMap map[string]any
	register   register.Register
	limiter    *rate.Limiter
}

type TcpConn struct {
	conn     net.Conn
	respChan chan *Response
}

func (c *TcpConn) Send(head []byte, body []byte) error {
	_, err := c.conn.Write(head)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(body)
	if err != nil {
		return err
	}
	return nil
}

func NewTcpServer(host string, port uint64) (*TcpServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	r := register.LoadRegister()
	if err := r.CreateClient(); err != nil {
		return nil, err
	}
	limiter := rate.NewLimiter(1, 1)
	return &TcpServer{
		host:     host,
		port:     port,
		listener: listener,
		register: r,
		limiter:  limiter,
	}, nil
}

func (s *TcpServer) Register(name string, service any) {
	t := reflect.TypeOf(service)
	if t.Kind() != reflect.Ptr {
		panic("service must be a pointer to struct")
	}
	name = fmt.Sprintf("rpc-%s", name)
	s.serviceMap[name] = service
	err := s.register.RegisterService(name, s.host, int(s.port))
	if err != nil {
		panic(err)
	}
}

func (s *TcpServer) Start() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		tcpConn := &TcpConn{conn: conn, respChan: make(chan *Response, 1)}
		go s.readHandler(tcpConn)
		go s.writeHandler(tcpConn)
	}
}

func (s *TcpServer) Stop() {
	err := s.listener.Close()
	if err != nil {
		panic(err)
	}
}

func (s *TcpServer) readHandler(conn *TcpConn) {
	defer func() {
		if r := recover(); r != nil {
			log.GWLogger.Error(r)
			err := conn.conn.Close()
			if err != nil {
				return
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := s.limiter.WaitN(ctx, 1)
	if err != nil {
		conn.respChan <- &Response{
			Code: 429,
			Msg:  "too many requests",
		}
		return
	}
	msg, err := decodeFrame(conn)
	if err != nil {
		conn.respChan <- &Response{
			Code: 500,
			Msg:  err.Error(),
		}
		return
	}
	if msg.Header.MsgType == msgRequest {
		req := msg.Data.(*Request)
		resp := &Response{
			RequestID:      req.RequestID,
			SerializerType: msg.Header.SerializerType,
			CompressorType: msg.Header.CompressorType,
		}
		serviceName := req.ServiceName
		service, ok := s.serviceMap[serviceName]
		if !ok {
			resp.Code = 404
			resp.Msg = "service not found: " + serviceName
			conn.respChan <- resp
			return
		}
		methodName := req.MethodName
		method, ok := reflect.TypeOf(service).MethodByName(methodName)
		if !ok {
			resp.Code = 404
			resp.Msg = "method not found: " + methodName
			conn.respChan <- resp
			return
		}
		args := req.Args
		var values []reflect.Value
		for _, arg := range args {
			values = append(values, reflect.ValueOf(arg))
		}
		resVal := method.Func.Call(values)
		res := make([]any, len(resVal))
		for i, v := range resVal {
			res[i] = v.Interface()
		}
		err, ok := res[len(res)-1].(error)
		if ok && err != nil {
			resp.Code = 500
			resp.Msg = err.Error()
			conn.respChan <- resp
			return
		}
		resp.Code = 200
		resp.Data = res[0]
		conn.respChan <- resp
		return
	}
}

func (s *TcpServer) writeHandler(conn *TcpConn) {
	select {
	case resp := <-conn.respChan:
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				panic(err)
			}
		}(conn.conn)
		msg := &Message{
			Header: &Header{
				MagicNumber:    MagicNumber,
				Version:        Version,
				MsgType:        msgResponse,
				CompressorType: resp.CompressorType,
				SerializerType: resp.SerializerType,
				RequestID:      resp.RequestID,
			},
			Data: resp,
		}
		head, body, err := encodeFrame(msg)
		if err != nil {
			// log error
			log.GWLogger.Error("failed to encode response: " + err.Error())
			return
		}
		err = conn.Send(head, body)
		if err != nil {
			// log error
			log.GWLogger.Error("failed to send response: " + err.Error())
			return
		}
	}
}

func encodeFrame(msg *Message) ([]byte, []byte, error) {
	headerBuf := make([]byte, 17)
	headerBuf[0] = msg.Header.MagicNumber
	headerBuf[1] = msg.Header.Version
	headerBuf[6] = byte(msg.Header.MsgType)
	headerBuf[7] = byte(msg.Header.CompressorType)
	headerBuf[8] = byte(msg.Header.SerializerType)
	binary.BigEndian.PutUint64(headerBuf[9:], uint64(msg.Header.RequestID))
	// serialize
	serializer := loadSerializer(msg.Header.SerializerType)
	sData, err := serializer.Serialize(msg.Data)
	if err != nil {
		return nil, nil, err
	}
	// compress
	compressor := loadCompressor(msg.Header.CompressorType)
	cData, err := compressor.Compress(sData)
	if err != nil {
		return nil, nil, err
	}
	binary.BigEndian.PutUint32(headerBuf[2:6], uint32(17+len(cData)))
	return headerBuf, cData, nil
}

func decodeFrame(conn *TcpConn) (*Message, error) {
	// header
	headerBuf := make([]byte, 17)
	_, err := io.ReadFull(conn.conn, headerBuf)
	if err != nil {
		return nil, err
	}
	magicNumber := headerBuf[0]
	if magicNumber != MagicNumber {
		return nil, errors.New("invalid magic number")
	}
	version := headerBuf[1]
	fullLength := int32(binary.BigEndian.Uint32(headerBuf[2:6]))
	messageType := MessageType(headerBuf[6])
	compressorType := CompressorProtocol(headerBuf[7])
	serializerType := SerializerProtocol(headerBuf[8])
	requestID := int64(binary.BigEndian.Uint64(headerBuf[9:]))
	// body
	bodyLength := fullLength - 17
	bodyBuf := make([]byte, bodyLength)
	_, err = io.ReadFull(conn.conn, bodyBuf)
	if err != nil {
		return nil, err
	}
	compressor := loadCompressor(compressorType)
	serializer := loadSerializer(serializerType)
	dBody, err := compressor.Decompress(bodyBuf)
	if err != nil {
		return nil, err
	}
	// deserialize
	msg := &Message{
		Header: &Header{
			MagicNumber:    magicNumber,
			Version:        version,
			FullLength:     fullLength,
			MsgType:        messageType,
			CompressorType: compressorType,
			SerializerType: serializerType,
			RequestID:      requestID,
		},
	}
	if messageType == msgRequest {
		req := &Request{}
		if err := serializer.Deserialize(dBody, req); err != nil {
			return nil, err
		}
		msg.Data = req
		return msg, nil
	}
	if messageType == msgResponse {
		resp := &Response{}
		if err := serializer.Deserialize(dBody, resp); err != nil {
			return nil, err
		}
		msg.Data = resp
		return msg, nil
	}
	return nil, errors.New("unknown message type")
}

type Client interface {
	Connect(string) error
	Invoke(context.Context, string, string, []any) (any, error)
	Close() error
}

type TcpClient struct {
	conn   *TcpConn
	option TcpClientOption
}

func (c *TcpClient) Connect(service string) error {
	service = fmt.Sprintf("rpc-%s", service)
	addr, err := c.option.Register.GetInstance(service)
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("tcp", addr, c.option.ConnectTimeout)
	if err != nil {
		return err
	}
	c.conn = &TcpConn{conn: conn, respChan: make(chan *Response, 1)}
	return nil
}

func (c *TcpClient) Invoke(_ context.Context, service string, method string, args []any) (any, error) {
	req := &Request{
		RequestID:   time.Now().UnixNano(),
		ServiceName: service,
		MethodName:  method,
		Args:        args,
	}
	msg := &Message{
		Header: &Header{
			MagicNumber:    MagicNumber,
			Version:        Version,
			MsgType:        msgRequest,
			CompressorType: c.option.CompressorType,
			SerializerType: c.option.SerializerType,
			RequestID:      req.RequestID,
		},
		Data: req,
	}
	header, body, err := encodeFrame(msg)
	if err != nil {
		return nil, err
	}
	err = c.conn.Send(header, body)
	if err != nil {
		return nil, err
	}
	go c.readHandler(c.conn.respChan)
	resp := <-c.conn.respChan
	return resp, nil
}

func (c *TcpClient) Close() error {
	if c.conn != nil {
		return c.conn.conn.Close()
	}
	return nil
}

func (c *TcpClient) readHandler(respChan chan *Response) {
	defer func() {
		if r := recover(); r != nil {
			log.GWLogger.Error(r)
			err := c.conn.conn.Close()
			if err != nil {
				return
			}
		}
	}()
	for {
		msg, err := decodeFrame(c.conn)
		if err != nil {
			resp := &Response{
				Code: 500,
				Msg:  err.Error(),
			}
			respChan <- resp
			return
		}
		if msg.Header.MsgType == msgResponse {
			resp := msg.Data.(*Response)
			respChan <- resp
			return
		}
	}
}

func NewTcpClient(option TcpClientOption) *TcpClient {
	return &TcpClient{
		option: option,
	}
}

type TcpClientOption struct {
	Retries        int
	ConnectTimeout time.Duration
	CompressorType CompressorProtocol
	SerializerType SerializerProtocol
	Register       register.Register
	Host           string
	Port           int
}

var DefaultTcpClientOption = TcpClientOption{
	Retries:        3,
	ConnectTimeout: 5 * time.Second,
	CompressorType: CompressorGzip,
	SerializerType: SerializerGob,
	Host:           "127.0.0.1",
	Port:           9090,
}

type TcpClientProxy struct {
	client *TcpClient
	option TcpClientOption
}

func NewTcpClientProxy(option TcpClientOption) *TcpClientProxy {
	return &TcpClientProxy{option: option}
}

func (p *TcpClientProxy) Call(ctx context.Context, service string, method string, args []any) (any, error) {
	client := NewTcpClient(p.option)
	client.option.Register = register.LoadRegister()
	err := client.option.Register.CreateClient()
	if err != nil {
		return nil, err
	}
	p.client = client
	err = client.Connect(service)
	if err != nil {
		return nil, err
	}
	for i := 0; i < p.option.Retries; i++ {
		res, err := client.Invoke(ctx, service, method, args)
		if err != nil {
			if i >= p.option.Retries-1 {
				log.GWLogger.Error(fmt.Sprintf("rpc call %s.%s failed after %d retries: %v", service, method, p.option.Retries, err))
				err := client.Close()
				if err != nil {
					return nil, err
				}
				return nil, err
			}
			continue
		}
		err = client.Close()
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	return nil, errors.New("unreachable code")
}

func loadCompressor(t CompressorProtocol) Compressor {
	switch t {
	case CompressorGzip:
		return &GzipCompressor{}
	default:
		return &GzipCompressor{}
	}
}

func loadSerializer(t SerializerProtocol) Serializer {
	switch t {
	case SerializerGob:
		return &GobSerializer{}
	default:
		return &GobSerializer{}
	}
}
