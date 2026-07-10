//go:build !darwin && !windows

package main

import (
	"errors"

	"github.com/eolseng/wow-markets/companion/internal/updatefeed"
)

type unsupportedPlatformUpdater struct{}

func newPlatformUpdater() platformUpdater { return unsupportedPlatformUpdater{} }

func platformUpdateTarget() (updatefeed.Target, string, bool) {
	return updatefeed.Target{}, "", false
}

func (unsupportedPlatformUpdater) Start(string) error {
	return errors.New("automatic updates are supported on macOS and Windows")
}
func (unsupportedPlatformUpdater) SetFeedURL(string) error { return nil }
func (unsupportedPlatformUpdater) Check() error            { return nil }
func (unsupportedPlatformUpdater) Install(string) error    { return nil }
func (unsupportedPlatformUpdater) Close()                  {}
func (unsupportedPlatformUpdater) ManagesDownloads() bool  { return false }
