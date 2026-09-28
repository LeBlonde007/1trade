package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

// warnOut is where deprecation warnings go (stderr, so --json output on stdout stays clean).
var warnOut io.Writer = os.Stderr

// successor pulls the model id out of the gateway's `Link: </v1/models/ID>; rel="successor-version"`.
var successor = regexp.MustCompile(`</v1/models/([^>]+)>;\s*rel="successor-version"`)

// warnDeprecation prints one warning when the gateway flags the model as deprecated (F10: the
// Deprecation and Sunset headers, plus a successor Link when there is a replacement).
func warnDeprecation(h http.Header) {
	if h.Get("Deprecation") == "" {
		return
	}
	msg := "warning: this model is deprecated"
	if t, err := http.ParseTime(h.Get("Sunset")); err == nil {
		msg += " and stops serving on " + t.UTC().Format(time.DateOnly)
	}
	if m := successor.FindStringSubmatch(h.Get("Link")); m != nil {
		msg += "; switch to " + m[1]
	}
	fmt.Fprintln(warnOut, msg)
}
