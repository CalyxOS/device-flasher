SPDX-FileCopyrightText: 2020 CIS Maxwell, LLC. All rights reserved.
SPDX-FileCopyrightText: The Calyx Institute
SPDX-License-Identifier: Apache-2.0

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

  This runs `make` in a container.

Development:
  Install pre-commit https://pre-commit.com and enable the git hooks, so
  that formatting, vet, staticcheck and tests are checked before each commit:

    pip install pre-commit
    pre-commit install

  Run all the checks on the whole tree at any time:

    pre-commit run --all-files

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

Test
