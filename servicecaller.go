package servicecaller

import (
	"bytes"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
)

func New() *ServiceCaller {
	return &ServiceCaller{
		server: rpc.NewServer(),
	}
}

type ServiceCaller struct {
	server     *rpc.Server
	serviceMap sync.Map
}

func (me *ServiceCaller) RegisterService(name string, rcvr any) error {
	if err := me.server.RegisterName(name, rcvr); err != nil {
		return err
	}
	me.serviceMap.Store(name, rcvr)
	return nil
}

func (me *ServiceCaller) Close() error {
	return nil
}

func (me *ServiceCaller) GetService(name string) any {
	svr, _ := me.serviceMap.Load(name)
	return svr
}

func (me *ServiceCaller) Call(serviceMethod string, args, reply any) error {
	done := <-me.Go(serviceMethod, args, reply).Done
	return done.Error
}

func (me *ServiceCaller) Go(serviceMethod string, args, reply any) *rpc.Call {
	reader := NewReadWriter(bytes.NewBuffer(nil))
	writer := NewReadWriter(bytes.NewBuffer(nil))
	defer func() {
		close(writer.readable)
		me.server.ServeCodec(jsonrpc.NewServerCodec(NewReadWriteCloser(writer, reader)))
		close(reader.readable)
	}()
	cli := jsonrpc.NewClient(NewReadWriteCloser(reader, writer))
	return cli.Go(serviceMethod, args, reply, make(chan *rpc.Call, 1))
}

type ReadWriteCloser struct {
	reader *ReadWriter
	writer *ReadWriter
}

func NewReadWriteCloser(reader, writer *ReadWriter) *ReadWriteCloser {
	return &ReadWriteCloser{
		reader: reader,
		writer: writer,
	}
}

func (me *ReadWriteCloser) Read(p []byte) (n int, err error) {
	return me.reader.Read(p)
}
func (me *ReadWriteCloser) Write(p []byte) (n int, err error) {
	return me.writer.Write(p)
}

func (me *ReadWriteCloser) Close() error {
	return nil
}

func NewReadWriter(buffer *bytes.Buffer) *ReadWriter {
	return &ReadWriter{
		buffer:   buffer,
		readable: make(chan struct{}, 1),
	}
}

type ReadWriter struct {
	buffer   *bytes.Buffer
	readable chan struct{}
}

func (me *ReadWriter) Read(p []byte) (n int, err error) {
	<-me.readable
	return me.buffer.Read(p)
}

func (me *ReadWriter) Write(p []byte) (n int, err error) {
	return me.buffer.Write(p)
}
