package main

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/debuglog"
	"github.com/gammons/slk/internal/ui"
)

// notifyClick is the payload a notification carries for its click: the
// macOS helper writes it back to slk's click socket verbatim.
type notifyClick struct {
	Team     string `json:"team"`
	Channel  string `json:"channel"`
	TS       string `json:"ts"`
	ThreadTS string `json:"thread_ts,omitempty"`
}

func (c notifyClick) encode() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// notifyClickSocket is this process's click socket. Per process, so a
// click reaches the slk that posted the notification.
func notifyClickSocket() string {
	return filepath.Join(xdgCache(), "notify-"+strconv.Itoa(os.Getpid())+".sock")
}

// listenNotifyClicks accepts clicks on path and hands each to send as a
// ui.NotificationClickedMsg. The returned func closes the listener and
// removes the socket.
func listenNotifyClicks(path string, send func(tea.Msg)) (func(), error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, err
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go readNotifyClick(conn, send)
		}
	}()
	return func() {
		ln.Close()
		_ = os.Remove(path)
	}, nil
}

func readNotifyClick(conn net.Conn, send func(tea.Msg)) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var c notifyClick
	if err := json.NewDecoder(io.LimitReader(conn, 4096)).Decode(&c); err != nil {
		debuglog.Notify("notification click: %v", err)
		return
	}
	send(ui.NotificationClickedMsg{TeamID: c.Team, ChannelID: c.Channel, TS: c.TS, ThreadTS: c.ThreadTS})
}
