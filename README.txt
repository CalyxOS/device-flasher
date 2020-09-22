Copyright © 2020 CIS Maxwell, LLC. All rights reserved.
Copyright © 2020 The Calyx Institute

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

Execution:
Plug each device of a same model to a USB port

The following files must be available in the current directory:
    CalyxOS factory image

 On Windows:
    Double-click on CalyxOS-flasher_windows.exe (will not show error output)
    or 
    Open PowerShell or Command Line
    Type: .\CalyxOS-flasher_windows.exe
    Press enter
 On Linux:
    Open a terminal in the current directory
    Type: sudo ./CalyxOS-flasher_linux
    Press enter
 On Mac:
    Open a terminal in the current directory
    Type: ./CalyxOS-flasher_darwin
    Press enter