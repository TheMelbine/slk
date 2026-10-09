package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui"
)

func TestListenNotifyClicks_DeliversClick(t *testing.T) {
	// A short dir: macOS caps unix socket paths at 104 bytes and
	// t.TempDir() paths there run long.
	dir, err := os.MkdirTemp("/tmp", "slk")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "n.sock")
	got := make(chan tea.Msg, 1)
	stop, err := listenNotifyClicks(path, func(m tea.Msg) { got <- m })
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer stop()

	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	click := notifyClick{Team: "T1", Channel: "C1", TS: "2.0", ThreadTS: "1.0"}
	if _, err := conn.Write([]byte(click.encode())); err != nil {
		t.Fatal(err)
	}
	conn.Close()

	select {
	case m := <-got:
		want := ui.NotificationClickedMsg{TeamID: "T1", ChannelID: "C1", TS: "2.0", ThreadTS: "1.0"}
		if m != want {
			t.Errorf("got %#v, want %#v", m, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no click delivered")
	}
}
