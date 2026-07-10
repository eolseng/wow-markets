//go:build darwin

package main

/*
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

int wow_sparkle_start(const char *feed_url);
int wow_sparkle_set_feed_url(const char *feed_url);
int wow_sparkle_check(void);
void wow_sparkle_close(void);
const char *wow_sparkle_last_error(void);
*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/eolseng/wow-markets/companion/internal/updatefeed"
)

type darwinPlatformUpdater struct{}

func newPlatformUpdater() platformUpdater { return darwinPlatformUpdater{} }

func platformUpdateTarget() (updatefeed.Target, string, bool) {
	return updatefeed.TargetMacOSARM64, "wow-markets-companion-macos-arm64.dmg", true
}

func (darwinPlatformUpdater) Start(feedURL string) error {
	cValue := C.CString(feedURL)
	defer C.free(unsafe.Pointer(cValue))
	if C.wow_sparkle_start(cValue) == 0 {
		return sparkleError()
	}
	return nil
}

func (darwinPlatformUpdater) SetFeedURL(feedURL string) error {
	cValue := C.CString(feedURL)
	defer C.free(unsafe.Pointer(cValue))
	if C.wow_sparkle_set_feed_url(cValue) == 0 {
		return sparkleError()
	}
	return nil
}

func (darwinPlatformUpdater) Check() error {
	if C.wow_sparkle_check() == 0 {
		return sparkleError()
	}
	return nil
}

func (darwinPlatformUpdater) Install(string) error { return darwinPlatformUpdater{}.Check() }
func (darwinPlatformUpdater) Close()               { C.wow_sparkle_close() }
func (darwinPlatformUpdater) ManagesDownloads() bool {
	return true
}

func sparkleError() error {
	message := C.GoString(C.wow_sparkle_last_error())
	if message == "" {
		message = "Sparkle updater operation failed"
	}
	return errors.New(message)
}
