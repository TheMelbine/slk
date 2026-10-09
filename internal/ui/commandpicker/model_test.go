package commandpicker

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var cmds = []Command{
	{Name: "/github", AppName: "GitHub", Desc: "Work with GitHub"},
	{Name: "/remind", Usage: "[what] [when]", Desc: "Set a reminder"},
	{Name: "/rename", Desc: "Rename a channel"},
	{Name: "/shrug"},
}

func names(cs []Command) string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return strings.Join(out, " ")
}

func TestFilter_PrefixFirstThenContains(t *testing.T) {
	m := New()
	m.SetCommands(cmds)
	m.Open()
	if got := names(m.Filtered()); got != "/github /remind /rename /shrug" {
		t.Fatalf("empty query = %q", got)
	}
	m.SetQuery("re")
	if got := names(m.Filtered()); got != "/remind /rename" {
		t.Errorf("re = %q", got)
	}
	m.SetQuery("hub")
	if got := names(m.Filtered()); got != "/github" {
		t.Errorf("hub = %q", got)
	}
	m.SetQuery("RU")
	if got := names(m.Filtered()); got != "/shrug" {
		t.Errorf("RU = %q", got)
	}
}

func TestSelect_FollowsCursor(t *testing.T) {
	m := New()
	m.SetCommands(cmds)
	m.Open()
	m.SetQuery("re")
	m.MoveDown()
	if c := m.Select(); c == nil || c.Name != "/rename" {
		t.Fatalf("Select = %+v", c)
	}
	m.SetQuery("zzz")
	if c := m.Select(); c != nil {
		t.Errorf("Select on no match = %+v", c)
	}
}

func TestView_ShowsUsageAndApp(t *testing.T) {
	m := New()
	m.SetCommands(cmds)
	m.Open()
	out := ansi.Strip(m.View(80))
	for _, want := range []string{"/remind [what] [when]", "Set a reminder", "GitHub · Work with GitHub"} {
		if !strings.Contains(out, want) {
			t.Errorf("view lacks %q:\n%s", want, out)
		}
	}
}
