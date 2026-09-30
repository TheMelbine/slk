package main

import (
	"errors"
	"strings"
	"testing"
)

// Shapes of "Copy as cURL" as Chrome and Firefox produce them, trimmed to what
// matters. The values are fake.
const (
	chromeCurl = `curl 'https://acme.slack.com/api/conversations.history?_x_id=abc&slack_route=T0123' \
  -H 'accept: */*' \
  -b 'b=abc123; d=xoxd-AbC%2Fdef%2BGhi%3D%3D; d-s=1700000000; lc=1700000000' \
  -H 'origin: https://app.slack.com' \
  --data-raw $'------WebKitFormBoundaryXYZ\r\nContent-Disposition: form-data; name="token"\r\n\r\nxoxc-1111-2222-3333-abcdef\r\n------WebKitFormBoundaryXYZ--\r\n'
`
	firefoxCurl = `curl 'https://acme.slack.com/api/client.counts' -X POST -H 'Content-Type: application/x-www-form-urlencoded' -H 'Cookie: d-s=17; d=xoxd-ZZZ%2F%3D; b=x' --data-raw 'token=xoxc-9999-8888-aa&_x_reason=x'
`
)

func TestParseBrowserSession(t *testing.T) {
	tests := []struct {
		name, paste, token, cookie string
	}{
		{"chrome", chromeCurl, "xoxc-1111-2222-3333-abcdef", "xoxd-AbC%2Fdef%2BGhi%3D%3D"},
		{"firefox, d-s before d", firefoxCurl, "xoxc-9999-8888-aa", "xoxd-ZZZ%2F%3D"},
		{"cookie first in the header", `-H 'cookie: d=xoxd-A1; b=2' token=xoxc-1-2`, "xoxc-1-2", "xoxd-A1"},
		{"bare token", "xoxc-1-2-3\n", "xoxc-1-2-3", ""},
		{"bare cookie", "  xoxd-Q%2F\n", "", "xoxd-Q%2F"},
		{"cookie with its name", "d=xoxd-Q\n", "", "xoxd-Q"},
		{"only d-s", `-b 'd-s=xoxd-nope' token=xoxc-1`, "xoxc-1", ""},
		{"the page, not an api call", `curl 'https://app.slack.com/client/T1' -b 'd=xoxd-A'`, "", "xoxd-A"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, cookie := parseBrowserSession(tc.paste)
			if token != tc.token || cookie != tc.cookie {
				t.Errorf("parseBrowserSession = (%q, %q), want (%q, %q)", token, cookie, tc.token, tc.cookie)
			}
		})
	}
}

func TestPasteComplete(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"\n\n", false},
		{"xoxc-1\n", true},
		{"curl 'x' \\\n", false},
		{"curl 'x' \\  \n", false},
		{"curl 'x' \\\n  -H 'a: b'\n", true},
	}
	for _, tc := range tests {
		if got := pasteComplete(tc.s); got != tc.want {
			t.Errorf("pasteComplete(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// A raw-mode terminal sends \r for Enter; a multi-line paste must be read up
// to the line that closes the command, and not past it.
func TestReadPasteMultiLineCurl(t *testing.T) {
	in := strings.ReplaceAll(chromeCurl, "\n", "\r") + "left over"
	got, err := readPaste(strings.NewReader(in))
	if err != nil {
		t.Fatalf("readPaste: %v", err)
	}
	if got != chromeCurl {
		t.Errorf("readPaste = %q, want %q", got, chromeCurl)
	}
}

// A line longer than the 4096-byte read buffer (and than the tty's canonical
// line limit) must come through whole.
func TestReadPasteLongLine(t *testing.T) {
	long := "curl -b 'd=xoxd-" + strings.Repeat("A", 10000) + "'\r"
	got, err := readPaste(strings.NewReader(long))
	if err != nil {
		t.Fatalf("readPaste: %v", err)
	}
	if want := strings.ReplaceAll(long, "\r", "\n"); got != want {
		t.Errorf("readPaste returned %d bytes, want %d", len(got), len(want))
	}
}

func TestReadPasteControlKeys(t *testing.T) {
	if _, err := readPaste(strings.NewReader("xoxc-1\x03")); !errors.Is(err, errPasteCancelled) {
		t.Errorf("Ctrl-C: err = %v, want errPasteCancelled", err)
	}
	got, err := readPaste(strings.NewReader("curl 'x' \\\r\x04more"))
	if err != nil || got != "curl 'x' \\\n" {
		t.Errorf("Ctrl-D: readPaste = (%q, %v), want what was read so far", got, err)
	}
	got, err = readPaste(strings.NewReader("xoxc-1"))
	if err != nil || got != "xoxc-1" {
		t.Errorf("EOF: readPaste = (%q, %v), want what was read so far", got, err)
	}
}
