package socks5

import (
	"bufio"
	"io"
	"net"
)

type connection struct {
	net.Conn
	reader  *bufio.Reader
	done    chan struct{}
	closeFn func()
}

func newConnection(conn net.Conn) *connection {
	return &connection{
		Conn:   conn,
		reader: bufio.NewReader(conn),
		done:   make(chan struct{}),
	}
}

func (c *connection) readByte() (byte, error) {
	return c.reader.ReadByte()
}

// read fills p completely: a field may arrive split across several TCP segments.
func (c *connection) read(p []byte) (int, error) {
	return io.ReadFull(c.reader, p)
}

// flushBuffered writes the bytes that the client sent right after the request
// and that are still held in the handshake reader.
func (c *connection) flushBuffered(dst io.Writer) (int64, error) {
	n := c.reader.Buffered()
	if n == 0 {
		return 0, nil
	}

	buffered, err := c.reader.Peek(n)
	if err != nil {
		return 0, err
	}

	written, err := dst.Write(buffered)

	return int64(written), err
}

func (c *connection) write(p []byte) (int, error) {
	return c.Conn.Write(p)
}

func (c *connection) isActive() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

func (c *connection) equalAddresses(address net.Addr) bool {
	currentHost, _, err := net.SplitHostPort(c.RemoteAddr().String())
	if err != nil {
		return false
	}

	incomingHost, _, err := net.SplitHostPort(address.String())
	if err != nil {
		return false
	}

	return currentHost == incomingHost
}

func (c *connection) onClose(f func()) {
	c.closeFn = f
}

func (c *connection) keepAlive() {
	io.Copy(io.Discard, c)

	if !c.isActive() {
		return
	}

	close(c.done)

	c.closeFn()
}
