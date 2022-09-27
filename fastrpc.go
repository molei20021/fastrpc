package fastrpc

import (
	"net"
	"syscall"
)

type Action int

const (
	None Action = iota
	Close
	Shutdown
)

type Event struct {
	LoopNum int
	Open    func(c Conn) (out []byte, opts Options, action Action)
	Close   func(c Conn, err error) (action Action)
	Data    func(c Conn, in []byte) (out []byte, action Action)
	Serve   func(svr Server) (action Action)
}

type Conn interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Server struct {
	Addr    net.Addr
	LoopNum int
}

func Serve(event Event, addr string) (err error) {
	var ln = new(listener)
	ln.ln, err = net.Listen("tcp", addr)
	ln.lnAddr = ln.ln.Addr()
	defer ln.ln.Close()
	switch netln := ln.ln.(type) {
	case *net.TCPListener:
		if ln.f, err = netln.File(); err != nil {
			ln.close()
			return
		}
		ln.fd = int(ln.f.Fd())
		if err = syscall.SetNonblock(ln.fd, true); err != nil {
			ln.close()
			return
		}
	}
	return serve(event, ln)
}
