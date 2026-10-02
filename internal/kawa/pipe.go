package kawa

import (
	"errors"
	"fmt"
	"syscall"

	"deedles.dev/wlr"
)

// loopPipe is a pipe whose read end the event loop watches. Other
// goroutines write to it to hand things to the event loop, which is
// the only place that may touch the server.
type loopPipe struct {
	r, w int
	src  wlr.EventSource
}

// newLoopPipe creates a pipe and has loop call cb whenever the pipe has
// something to read. The read end is non-blocking so that cb can drain
// it.
func newLoopPipe(loop wlr.EventLoop, cb func(fd uintptr, mask wlr.EventMask)) (loopPipe, error) {
	var fds [2]int
	err := syscall.Pipe2(fds[:], syscall.O_CLOEXEC)
	if err != nil {
		return loopPipe{}, fmt.Errorf("create pipe: %w", err)
	}

	err = syscall.SetNonblock(fds[0], true)
	if err != nil {
		syscall.Close(fds[0])
		syscall.Close(fds[1])
		return loopPipe{}, fmt.Errorf("set pipe non-blocking: %w", err)
	}

	src := loop.AddFd(uintptr(fds[0]), wlr.EventReadable, cb)
	if !src.Valid() {
		syscall.Close(fds[0])
		syscall.Close(fds[1])
		return loopPipe{}, errors.New("add pipe to event loop")
	}

	return loopPipe{r: fds[0], w: fds[1], src: src}, nil
}

// close removes the pipe from the event loop and closes both ends. It
// does nothing if the pipe was never created or is already closed.
func (p *loopPipe) close() {
	if !p.src.Valid() {
		return
	}

	p.src.Remove()
	syscall.Close(p.r)
	syscall.Close(p.w)
	*p = loopPipe{}
}
