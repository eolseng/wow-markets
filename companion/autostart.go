package main

import "os"

const backgroundLaunchArgument = "--background"

func launchedInBackground() bool {
	return hasBackgroundLaunchArgument(os.Args[1:])
}

func hasBackgroundLaunchArgument(arguments []string) bool {
	for _, argument := range arguments {
		if argument == backgroundLaunchArgument {
			return true
		}
	}
	return false
}

func shouldRefreshLaunchAtLogin(enabled, backgroundLaunch bool) bool {
	return enabled && !backgroundLaunch
}
