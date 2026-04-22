package client

import (
	"net"
	"os"
)

// localConn is what the Unix socket yields. Trivial wrapper so tests can swap.
type localConn = net.Conn

type unixListener struct {
	net.Listener
	path string
}

func (u *unixListener) Close() error {
	err := u.Listener.Close()
	_ = os.Remove(u.path)
	return err
}

func listenUnix(path string) (net.Listener, error) {
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0o660)
	return &unixListener{Listener: ln, path: path}, nil
}
