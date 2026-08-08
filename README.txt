Copyright © 2020 CIS Maxwell, LLC. All rights reserved.
Copyright © 2020-2026 The Calyx Institute

Build:
Install Go on your machine https://golang.org/doc/install

  On Bash:
    GOPATH="path-to-flasher-source" GOOS=[darwin|linux|windows] GOARCH=amd64 go build -o CalyxOS-flasher_[darwin|linux|windows.exe]
  On Cmd:
    SET GOPATH="path-to-flasher-source"
    SET GOOS=[darwin|linux|windows]
    SET GOARCH=amd64
    go build -o CalyxOS-flasher_[darwin|linux|windows.exe]
  On PowerShell:
    $Env:GOPATH="path-to-flasher-source"; $Env:GOOS = "[darwin|linux|windows]"; $Env:GOARCH = "amd64"; go build -o CalyxOS-flasher_[darwin|linux|windows.exe]

Build with Docker:
  ./build.sh

  This runs `make` in a container and prints SHA-256 checksums of the
  binaries. The macOS `device-flasher.darwin` is a universal binary with both
  Intel and Apple Silicon slices, assembled by konoui/lipo.

Execution:
Plug each device of a same model to a USB port

The following files must be available in the current directory:
    CalyxOS factory image

 On Windows:
    Double-click on device-flasher.exe (will not show error output)
    or
    Open PowerShell or Command Line
    Type: .\device-flasher.exe
    Press enter
 On Linux:
    Open a terminal in the current directory
    Type: sudo ./device-flasher.linux
    Press enter
 On Mac:
    Open a terminal in the current directory
    Type: ./device-flasher.darwin
    Press enter
