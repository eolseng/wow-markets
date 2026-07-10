//go:build windows

package main

import (
	"strings"
	"testing"
)

func TestWindowsStartupCommandShape(t *testing.T) {
	executable := `C:\Program Files\WoW Markets\WoW Markets Companion.exe`
	command, err := windowsStartupCommand(executable)
	if err != nil {
		t.Fatalf("windowsStartupCommand() error = %v", err)
	}
	if !strings.HasPrefix(command, `"C:\Program Files`) || !strings.HasSuffix(command, backgroundLaunchArgument) {
		t.Fatalf("startup command = %q", command)
	}
	if _, err := windowsStartupCommand(`C:\` + strings.Repeat("nested\\", 40) + `companion.exe`); err == nil {
		t.Fatal("windowsStartupCommand() accepted a command longer than the Run key limit")
	}
}
