package internal

import "syscall"

type Poll struct {
	epollFd int
	wakeFd  int
}

func CreateLoopPoll() (p *Poll) {
	var (
		err error
	)
	p = new(Poll)
	if p.epollFd, err = syscall.EpollCreate1(0); err != nil {
		panic(err)
	}
	r0, _, e0 := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if e0 != 0 {
		syscall.Close(p.epollFd)
		panic(err)
	}
	p.wakeFd = int(r0)
	p.AddLoopRead(p.wakeFd)
	return
}

func (p *Poll) AddLoopRead(wakeFd int) {
	if err := syscall.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, wakeFd,
		&syscall.EpollEvent{
			Fd:     int32(wakeFd),
			Events: syscall.EPOLLIN,
		}); err != nil {
		panic(err)
	}
}

func (p *Poll) LoopClose() (err error) {
	if err = syscall.Close(p.wakeFd); err != nil {
		return
	}
	if err = syscall.Close(p.epollFd); err != nil {
		return
	}
	return
}

func (p *Poll) LoopWait(execEpollFd func(connFd int) error) (err error) {
	var n int
	epollEvents := make([]syscall.EpollEvent, 64)
	for {
		n, err = syscall.EpollWait(p.epollFd, epollEvents, 100)
		if err != nil && err != syscall.EINTR {
			return
		}
		for i := 0; i < n; i++ {
			connFd := epollEvents[i].Fd
			if connFd == int32(p.wakeFd) {
				continue
			}
			if err = execEpollFd(int(connFd)); err != nil {
				return
			}
		}
	}
}

func (p *Poll) AddLoopReadWrite(wakeFd int) {
	if err := syscall.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, wakeFd,
		&syscall.EpollEvent{
			Fd:     int32(wakeFd),
			Events: syscall.EPOLLIN | syscall.EPOLLOUT,
		}); err != nil {
		panic(err)
	}
}

func (p *Poll) ModLoopRead(wakeFd int) {
	if err := syscall.EpollCtl(p.epollFd, syscall.EPOLL_CTL_MOD, wakeFd,
		&syscall.EpollEvent{
			Fd:     int32(wakeFd),
			Events: syscall.EPOLLIN,
		}); err != nil {
		panic(err)
	}
}

func (p *Poll) ModLoopReadWrite(wakeFd int) {
	if err := syscall.EpollCtl(p.epollFd, syscall.EPOLL_CTL_MOD, wakeFd,
		&syscall.EpollEvent{
			Fd:     int32(wakeFd),
			Events: syscall.EPOLLIN | syscall.EPOLLOUT,
		}); err != nil {
		panic(err)
	}
}

func SetKeepAlive(fd, sec int) (err error) {
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1); err != nil {
		return
	}
	if err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, sec); err != nil {
		return
	}
	if err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, sec); err != nil {
		return
	}
	return
}
