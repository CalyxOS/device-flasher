// SPDX-FileCopyrightText: 2020 CIS Maxwell, LLC. All rights reserved.
// SPDX-FileCopyrightText: The Calyx Institute
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func init() {
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32

	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
