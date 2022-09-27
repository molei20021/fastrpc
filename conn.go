package fastrpc

import (
	"net"
	"syscall"
)

type conn struct {
	fd         int
	loop       *loop
	sa         syscall.Sockaddr
	wBuffer    []byte
	opened     bool
	localAddr  net.Addr
	remoteAddr net.Addr
	action     Action
}

func (c *conn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
