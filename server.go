package fastrpc

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/molei20021/fastrpc/internal"
)

type server struct {
	event             Event
	loops             []*loop
	ln                *listener
	wg                sync.WaitGroup
	accepted          uintptr
	localAddrListener net.Addr
}

type loop struct {
	idx     int
	poll    *internal.Poll
	packets []byte
	fdconns map[int]*conn
	count   int32
}

type listener struct {
	ln     net.Listener
	fd     int
	lnAddr net.Addr
	f      *os.File
}

func (ln *listener) close() {
	if ln.fd != 0 {
		syscall.Close(ln.fd)
	}
	if ln.f != nil {
		ln.f.Close()
	}
	if ln.ln != nil {
		ln.ln.Close()
	}
}

var (
	errClosing = fmt.Errorf("closing")
)

func serve(event Event, ln *listener) (err error) {
	loopNum := event.LoopNum
	if loopNum == 0 {
		loopNum = runtime.NumCPU()
	}
	svr := &server{}
	svr.event = event
	svr.ln = ln

	if svr.event.Serve != nil {
		var svrTmp Server
		svrTmp.LoopNum = loopNum
		svrTmp.Addr = ln.lnAddr
		action := svr.event.Serve(svrTmp)
		switch action {
		case None:
		case Shutdown:
			return nil
		}
	}

	defer func() {
		svr.wg.Wait()

		for _, l := range svr.loops {
			for _, c := range l.fdconns {
				loopCloseConn(svr, l, c, nil)
			}
			l.poll.LoopClose()
		}
	}()

	for i := 0; i < loopNum; i++ {
		l := &loop{
			idx:     i,
			poll:    internal.CreateLoopPoll(),
			packets: make([]byte, 0xFFFF),
			fdconns: make(map[int]*conn),
		}
		l.poll.AddLoopRead(ln.fd)
		svr.loops = append(svr.loops, l)
	}
	svr.wg.Add(len(svr.loops))
	for _, l := range svr.loops {
		go loopRun(svr, l)
	}
	return
}

func loopRun(svr *server, l *loop) {
	defer func() {
		svr.wg.Done()
	}()
	l.poll.LoopWait(func(connFd int) (err error) {
		c := l.fdconns[connFd]
		switch {
		case c == nil:
			return loopAccept(svr, l, connFd)
		case !c.opened:
			return loopOpened(svr, l, c)
		case len(c.wBuffer) > 0:
			return loopWrite(svr, l, c)
		default:
			return loopRead(svr, l, c)
		}
	})
}

func loopAccept(svr *server, l *loop, fd int) (err error) {
	var (
		sa  syscall.Sockaddr
		nfd int
	)
	if svr.ln.fd == fd {
		if len(svr.loops) > 1 {
			idx := int(atomic.LoadUintptr(&svr.accepted)) % len(svr.loops)
			if idx != l.idx {
				return
			}
			atomic.AddUintptr(&svr.accepted, 1)
		}
		nfd, sa, err = syscall.Accept(fd)
		if err != nil && err != syscall.EAGAIN {
			return
		}
		if err = syscall.SetNonblock(nfd, true); err != nil {
			return
		}
		c := &conn{fd: nfd, sa: sa, loop: l, wBuffer: nil}
		l.fdconns[c.fd] = c
		l.poll.AddLoopReadWrite(c.fd)
		atomic.AddInt32(&l.count, 1)
	}
	return
}

func loopOpened(svr *server, l *loop, c *conn) (err error) {
	var (
		out    []byte
		opts   Options
		action Action
	)
	c.opened = true
	c.localAddr = svr.localAddrListener
	c.remoteAddr = internal.SockaddrToAddr(c.sa)
	if svr.event.Open != nil {
		out, opts, action = svr.event.Open(c)
		if len(out) > 0 {
			c.wBuffer = append(c.wBuffer, out...)
		}
		c.action = action
		if opts.KeepAliveSec > 0 {
			internal.SetKeepAlive(c.fd, opts.KeepAliveSec)
		}
	}
	return
}

func loopWrite(svr *server, l *loop, c *conn) (err error) {
	var n int
	n, err = syscall.Write(c.fd, c.wBuffer)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return loopCloseConn(svr, l, c, err)
	}
	if n == len(c.wBuffer) {
		if cap(c.wBuffer) > 4096 {
			c.wBuffer = nil
		} else {
			c.wBuffer = c.wBuffer[:0]
		}
	} else {
		c.wBuffer = c.wBuffer[n:]
	}
	if len(c.wBuffer) == 0 && c.action == None {
		l.poll.ModLoopRead(c.fd)
	}
	return
}

func loopCloseConn(svr *server, l *loop, c *conn, err error) error {
	atomic.AddInt32(&l.count, -1)
	delete(l.fdconns, c.fd)
	syscall.Close(c.fd)
	if svr.event.Close != nil {
		switch svr.event.Close(c, err) {
		case None:
		case Shutdown:
			return errClosing
		}
	}
	return nil
}

func loopRead(svc *server, l *loop, c *conn) (err error) {
	var (
		in     []byte
		out    []byte
		action Action
	)
	n, err := syscall.Read(c.fd, l.packets)
	if n == 0 || err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return loopCloseConn(svc, l, c, err)
	}
	in = l.packets[:n]
	if svc.event.Data != nil {
		out, action = svc.event.Data(c, in)
		c.action = action
		if len(out) > 0 {
			c.wBuffer = append(c.wBuffer[:0], out...)
		}
	}
	if len(c.wBuffer) > 0 || c.action != None {
		l.poll.ModLoopReadWrite(c.fd)
	}
	return
}
