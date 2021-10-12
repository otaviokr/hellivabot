package hbot

import (
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/ftrvxmtrx/fd"
)

// StartUnixListener starts up a unix domain socket listener for reconnects to
// be sent through
func (bot *Bot) StartUnixListener() {
	unaddr, err := net.ResolveUnixAddr("unix", bot.unixastr)
	if err != nil {
		panic(err)
	}

	list, err := net.ListenUnix("unix", unaddr)
	if err != nil {
		panic(err)
	}
	defer list.Close()
	bot.unixlist = list

	con, err := list.AcceptUnix()
	if err != nil {
		fmt.Println("unix listener error: ", err)
		return
	}
	defer con.Close()

	fi, err := bot.con.(*net.TCPConn).File()
	if err != nil {
		panic(err)
	}

	err = fd.Put(con, fi)
	if err != nil {
		panic(err)
	}

	select {
	case <-bot.Incoming:
	default:
		close(bot.Incoming)
	}
	close(bot.outgoing)
}

// Attempt to hijack session previously running bot
func (bot *Bot) hijackSession() bool {
	con, err := net.Dial("unix", bot.unixastr)
	if err != nil {
		bot.Info("Couldnt restablish connection, no prior bot.", "err", err)
		return false
	}
	defer con.Close()

	ncon, err := fd.Get(con.(*net.UnixConn), 1, nil)
	if err != nil {
		panic(err)
	}
	defer ncon[0].Close()

	netcon, err := net.FileConn(ncon[0])
	if err != nil {
		panic(err)
	}
	bot.reconnecting = true
	bot.con = netcon
	return true
}

// IsConnected hecks if connection is still open
func (bot *Bot) IsConnected() error {
	var sysErr error = nil
	rc, err := bot.con.(syscall.Conn).SyscallConn()
	if err != nil {
		bot.Errorf("Bot is not connected: %v", err)
		return err
	}

	err = rc.Read(func(fd uintptr) bool {
		var buf []byte = []byte{0}
		n, _, err := syscall.Recvfrom(int(fd), buf, syscall.MSG_PEEK | syscall.MSG_DONTWAIT)
		switch {
		case n == 0 && err == nil:
			sysErr = io.EOF

		case err == syscall.EAGAIN || err == syscall.EWOULDBLOCK:
			sysErr = nil

		default:
			sysErr = err
		}

		return true
	})
	if err != nil {
		return err
	}

	return sysErr
}