//go:build darwin

package main

import (
	"strings"
	"testing"
)

func TestLaunchAgentPayloadEscapesExecutableAndStartsHidden(t *testing.T) {
	payload := string(launchAgentPayload(`/Applications/WoW & Markets.app/Contents/MacOS/WoW Markets Companion`))
	for _, expected := range []string{
		launchAgentLabel,
		`/Applications/WoW &amp; Markets.app/Contents/MacOS/WoW Markets Companion`,
		backgroundLaunchArgument,
		"<key>RunAtLoad</key>",
	} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("launchAgentPayload() missing %q:\n%s", expected, payload)
		}
	}
}
