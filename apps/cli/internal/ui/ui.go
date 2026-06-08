// Package ui is the exascale CLI's presentation layer (F24): semantic colour, spinners, confirmations,
// and actionable error hints. It stays institutional (Bloomberg, not flashy) — green ▲ / red ▼, mono
// restraint. Colour auto-disables off-TTY and under NO_COLOR, so piped/CI output is plain and stable.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// colorOn reports whether ANSI styling should be emitted: a real terminal on stdout, and NO_COLOR unset.
var colorOn = os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))

// SetColor forces colour on/off (tests, --no-color). Returns the previous value.
func SetColor(on bool) bool { prev := colorOn; colorOn = on; return prev }

// wrap applies an ANSI SGR code, or returns s unchanged when colour is off.
func wrap(code, s string) string {
	if !colorOn {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// Bold, Dim, Green, Red, Cyan style a string (no-ops when colour is off).
func Bold(s string) string  { return wrap("1", s) }
func Dim(s string) string   { return wrap("2", s) }
func Green(s string) string { return wrap("32", s) }
func Red(s string) string   { return wrap("31", s) }
func Cyan(s string) string  { return wrap("36", s) }

// Delta renders a signed number as a credit movement: green ▲ for ≥0, red ▼ for <0, with the magnitude.
func Delta(n float64) string {
	if n < 0 {
		return Red(fmt.Sprintf("▼ %s", trim(-n)))
	}
	return Green(fmt.Sprintf("▲ %s", trim(n)))
}

// trim formats a float without trailing zeros (e.g. 12.50 → "12.5", 5 → "5").
func trim(n float64) string {
	s := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", n), "0"), ".")
	if s == "" || s == "-" {
		s = "0"
	}
	return s
}

// Confirm prompts "<prompt> [y/N]" on stderr and returns true only for y/yes. Non-TTY stdin returns
// false (callers pass --yes to skip), so a piped/CI run never blocks on a destructive prompt.
func Confirm(prompt string) bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	fmt.Fprintf(os.Stderr, "%s %s ", prompt, Dim("[y/N]"))
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// Spinner is a minimal in-flight indicator on stderr; it's a no-op off-TTY (so logs stay clean).
type Spinner struct {
	stop chan struct{}
	done chan struct{}
}

// StartSpinner shows an animated "<label>…" on stderr until Stop is called. Off-TTY it does nothing.
func StartSpinner(label string) *Spinner {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return &Spinner{}
	}
	s := &Spinner{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		defer close(s.done)
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				fmt.Fprint(os.Stderr, "\r\033[K") // clear the line
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s", Cyan(string(frames[i%len(frames)])), Dim(label))
				time.Sleep(90 * time.Millisecond)
			}
		}
	}()
	return s
}

// Stop ends the spinner and clears its line.
func (s *Spinner) Stop() {
	if s.stop == nil {
		return
	}
	close(s.stop)
	<-s.done
}

// Hint maps an upstream error (code/status) to the next action the user should take — the difference
// between a dead end and a self-service fix. Empty when there's no obvious next step.
func Hint(status int, code string) string {
	switch {
	case status == 401 || code == "invalid_token" || code == "unauthorized":
		return "run `exascale login` to authenticate"
	case status == 402 || code == "insufficient_credits":
		return "top up at the web app (Wallet → Buy credits), then retry"
	case code == "email_unverified":
		return "verify your email (check your inbox), then `exascale login`"
	case status == 403 || code == "forbidden":
		return "your account lacks permission for this — check your role/KYC status"
	case status == 404:
		return "not found — check the id/model with `exascale catalog` or `exascale gpu list`"
	case status == 429:
		return "rate limited — wait a moment and retry"
	case status >= 500:
		return "the service had an error — retry shortly; if it persists, check status"
	}
	return ""
}
