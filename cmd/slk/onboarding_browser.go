package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"charm.land/huh/v2"
	"golang.org/x/term"

	slackclient "github.com/gammons/slk/internal/slack"
)

// The browser-session flow signs a workspace in without the Slack desktop app
// (containers, servers, SSH sessions, or a browser-only user). It needs the
// same two values the desktop flow reads: the xoxc token and the d cookie.
// The d cookie is HttpOnly, so no DevTools console snippet can read it, but a
// "Copy as cURL" of any request to a Slack /api/ endpoint carries both. The
// user pastes that; a bare xoxc token followed by the d cookie also works.
//
// Nothing re-mints such a token: startup keeps the cached one when it cannot
// read a desktop cookie (see remintTokens), so it lasts as long as the
// browser session it came from.

var (
	errPasteCancelled = errors.New("paste cancelled")
	errNoToken        = errors.New("no xoxc- token found: copy a request to /api/, not the page itself")
	errNoCookie       = errors.New("no d cookie (xoxd-) found: copy the request from a browser signed in to app.slack.com")

	xoxcPattern = regexp.MustCompile(`xoxc-[A-Za-z0-9-]+`)
	// The d cookie, not d-s or any other cookie ending in d.
	dCookiePattern = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])d=(xoxd-[^;\s"'\\]+)`)
	// A value pasted on its own, without the "d=" prefix.
	bareCookiePattern = regexp.MustCompile(`^xoxd-[^;\s"'\\]+$`)
)

const browserSessionSteps = `No Slack desktop app needed: sign in from your browser instead.

  1. Open https://app.slack.com in your browser and sign in.
  2. DevTools > Network, filter on api/, click a channel, then right click
     one of the requests > Copy > Copy as cURL (bash).
  3. Paste it below (nothing is shown while you paste).

A bare xoxc- token works too; slk then asks for the d cookie
(DevTools > Application > Cookies > https://app.slack.com).`

// parseBrowserSession pulls the xoxc token and the d cookie out of pasted
// text. Either may be empty; the caller decides what to ask for next.
func parseBrowserSession(paste string) (token, cookie string) {
	token = xoxcPattern.FindString(paste)
	if m := dCookiePattern.FindStringSubmatch(paste); m != nil {
		cookie = m[1]
	} else if trimmed := strings.TrimSpace(paste); bareCookiePattern.MatchString(trimmed) {
		cookie = trimmed
	}
	return token, cookie
}

// pasteComplete reports whether the text read so far ends a pasted command:
// its last line is not empty and does not end with a shell continuation.
func pasteComplete(s string) bool {
	s = strings.TrimRight(s, "\n")
	if strings.TrimSpace(s) == "" {
		return false
	}
	last := strings.TrimRight(s[strings.LastIndexByte(s, '\n')+1:], " \t")
	return !strings.HasSuffix(last, `\`)
}

// readPaste reads a paste byte by byte until it is complete (see
// pasteComplete), or until Ctrl-D. Ctrl-C cancels. It expects a terminal in
// raw mode: in canonical mode Linux drops anything past 4095 bytes on a line,
// and a copied cURL line with its cookies easily goes past that.
func readPaste(r io.Reader) (string, error) {
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		for _, c := range buf[:n] {
			switch c {
			case 3: // Ctrl-C
				return "", errPasteCancelled
			case 4: // Ctrl-D
				return b.String(), nil
			case '\r':
				c = '\n'
			}
			b.WriteByte(c)
			if c == '\n' && pasteComplete(b.String()) {
				return b.String(), nil
			}
		}
		if errors.Is(err, io.EOF) {
			return b.String(), nil
		}
		if err != nil {
			return "", err
		}
	}
}

// readSecret prints prompt and reads a paste without echoing it. From a pipe
// it reads everything.
func readSecret(in *os.File, prompt string) (string, error) {
	fmt.Print(prompt)
	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		b, err := io.ReadAll(in)
		fmt.Println()
		return string(b), err
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	s, err := readPaste(in)
	_ = term.Restore(fd, old)
	fmt.Println()
	return s, err
}

// offerBrowserFallback is called when the desktop session cannot be read. On
// a terminal it offers the browser-session flow; otherwise it returns the
// desktop error unchanged.
func offerBrowserFallback(tokenStore *slackclient.TokenStore, st onboardingStyles, desktopErr error) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return desktopErr
	}
	useBrowser := true
	confirm := huh.NewConfirm().
		Title("Sign in with a browser session instead?").
		Affirmative("Yes").
		Negative("No").
		Value(&useBrowser)
	if err := huh.NewForm(huh.NewGroup(confirm)).WithTheme(huh.ThemeFunc(huh.ThemeDracula)).Run(); err != nil || !useBrowser {
		return desktopErr
	}
	return addWorkspaceFromBrowser(tokenStore, st)
}

// addWorkspaceFromBrowser runs the browser-session flow: read the paste,
// check the pair with auth.test, save the token like the desktop flow does.
func addWorkspaceFromBrowser(tokenStore *slackclient.TokenStore, st onboardingStyles) error {
	fmt.Println()
	fmt.Println(browserSessionSteps)
	fmt.Println()

	paste, err := readSecret(os.Stdin, "cURL command or xoxc token: ")
	if err != nil {
		return err
	}
	token, cookie := parseBrowserSession(paste)
	if token == "" {
		fmt.Println(st.errorText.Render("  " + errNoToken.Error()))
		return errNoToken
	}
	if cookie == "" {
		paste, err = readSecret(os.Stdin, "d cookie: ")
		if err != nil {
			return err
		}
		if _, cookie = parseBrowserSession(paste); cookie == "" {
			fmt.Println(st.errorText.Render("  " + errNoCookie.Error()))
			return errNoCookie
		}
	}

	fmt.Println(st.step.Render("Connecting..."))
	client := slackclient.NewClient(token, cookie)
	if err := client.Connect(context.Background()); err != nil {
		fmt.Println(st.errorText.Render(fmt.Sprintf("  Authentication failed: %v", err)))
		return fmt.Errorf("authentication failed: %w", err)
	}
	tok := slackclient.Token{
		AccessToken: token,
		Cookie:      cookie,
		Domain:      client.TeamSubdomain(),
		TeamID:      client.TeamID(),
		TeamName:    client.TeamName(),
	}
	if err := saveWorkspace(tokenStore, tok, st); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(st.dim.Render("  This session lasts as long as the browser's. When Slack signs it out, run"))
	fmt.Println(st.dim.Render("  slk --add-workspace --browser again."))
	fmt.Println(st.dim.Render("  Run ") + st.step.Render("slk") + st.dim.Render(" to start."))
	fmt.Println()
	return nil
}
