// +build windows

package main

import (
	"golang.org/x/sys/windows"
)

func init() {
	setConsoleMode(windows.Stdout, windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	setConsoleMode(windows.Stderr, windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}

func setConsoleMode(handle windows.Handle, flags uint32) {
	var mode uint32

	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		log.Printf("failed to get windows console mode for cli: %v\n", err)
	} else {
		err := windows.SetConsoleMode(handle, mode|flags)
		if err != nil {
			log.Printf("failed to set windows console mode for cli: %v\n", err)
		}
	}
}
