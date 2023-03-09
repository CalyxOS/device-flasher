// Copyright 2020 CIS Maxwell, LLC. All rights reserved.
// Copyright 2020 The Calyx Institute
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var input string

var executable, _ = os.Executable()
var cwd = filepath.Dir(executable)

var adb *exec.Cmd
var fastboot *exec.Cmd

var platformToolsZip string

var deviceFactoryFolderMap map[string]string

// Set via LDFLAGS, check Makefile
var version string

const OS = runtime.GOOS
const PLATFORM_TOOLS_VERSION = "33.0.3"

var (
	Error = Red
	Warn  = Yellow
)

var (
	Blue   = Color("\033[1;34m%s\033[0m")
	Red    = Color("\033[1;31m%s\033[0m")
	Yellow = Color("\033[1;33m%s\033[0m")
)

func Color(color string) func(...interface{}) string {
	return func(args ...interface{}) string {
		return fmt.Sprintf(color,
			fmt.Sprint(args...))
	}
}

func errorln(err interface{}, fatal bool) {
	log, _ := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	_, _ = fmt.Fprintln(log, err)
	_, _ = fmt.Fprintln(os.Stderr, Error(err))
	log.Close()
	if fatal {
		fmt.Println("Press enter to exit.")
		_, _ = fmt.Scanln(&input)
		os.Exit(1)
	}
}

func warnln(warning interface{}) {
	fmt.Println(Warn(warning))
}

func main() {
	_ = os.Remove("error.log")
	fmt.Println("Android Factory Image Flasher version " + version)
	// Map device codenames to their corresponding extracted factory image folders
	deviceFactoryFolderMap = getFactoryFolders()
	if len(deviceFactoryFolderMap) < 1 {
		errorln(errors.New("Cannot continue without a device factory image. Exiting..."), true)
	}
	err := getPlatformTools()
	if err != nil {
		errorln("Cannot continue without Android platform tools. Exiting...", false)
		errorln(err, true)
	}
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "start-server")
	err = platformToolCommand.Run()
	if err != nil {
		errorln("Cannot start ADB server", false)
		errorln(err, true)
	}
	warnln("1. Connect to a wifi network and ensure that no SIM cards are installed")
	warnln("2. Enable Developer Options on device (Settings -> About Phone -> tap \"Build number\" 7 times)")
	warnln("3. Enable USB debugging on device (Settings -> System -> Advanced -> Developer Options) and allow the computer to debug (hit \"OK\" on the popup when USB is connected)")
	warnln("4. Enable OEM Unlocking (in the same Developer Options menu)")
	fmt.Println()
	fmt.Print(Warn("Press ENTER to continue"))
	_, _ = fmt.Scanln(&input)
	fmt.Println()
	// Map serial numbers to device codenames by extracting them from adb and fastboot command output
	devices := getDevices()
	if len(devices) == 0 {
		errorln(errors.New("No devices to be flashed. Exiting..."), true)
	} else if !PARALLEL && len(devices) > 1 {
		errorln(errors.New("More than one device detected. Exiting..."), true)
	}
	fmt.Println()
	fmt.Println("Devices to be flashed: ")
	for serialNumber, device := range devices {
		fmt.Println(device + " " + serialNumber)
	}
	fmt.Println()
	fmt.Print(Warn("Press ENTER to continue"))
	_, _ = fmt.Scanln(&input)
	// Sequence: unlock bootloader -> execute flash-all script -> relock bootloader
	flashDevices(devices)
}

func getFactoryFolders() map[string]string {
	files, err := ioutil.ReadDir(cwd)
	if err != nil {
		errorln(err, true)
	}
	deviceFactoryFolderMap := map[string]string{}
	for _, file := range files {
		file := file.Name()
		if strings.Contains(file, "factory") && strings.HasSuffix(file, ".zip") {
			extracted, err := extractZip(path.Base(file), cwd)
			if err != nil {
				errorln("Cannot continue without a factory image. Exiting...", false)
				errorln(err, true)
			}
			device := strings.Split(file, "-")[0]
			if _, exists := deviceFactoryFolderMap[device]; !exists {
				deviceFactoryFolderMap[device] = extracted[0]
			} else {
				errorln("More than one factory image available for "+device, true)
			}
		}
	}
	return deviceFactoryFolderMap
}

func getPlatformTools() error {
	plaformToolsUrlMap := map[[2]string]string{
		[2]string{"darwin", "33.0.3"}:  "https://dl.google.com/android/repository/platform-tools_r33.0.3-darwin.zip",
		[2]string{"linux", "33.0.3"}:   "https://dl.google.com/android/repository/platform-tools_r33.0.3-linux.zip",
		[2]string{"windows", "33.0.3"}: "https://dl.google.com/android/repository/platform-tools_r33.0.3-windows.zip",
	}
	platformToolsChecksumMap := map[[2]string]string{
		[2]string{"darwin", "33.0.3"}:  "84acbbd2b2ccef159ae3e6f83137e44ad18388ff3cc66bb057c87d761744e595",
		[2]string{"linux", "33.0.3"}:   "ab885c20f1a9cb528eb145b9208f53540efa3d26258ac3ce4363570a0846f8f7",
		[2]string{"windows", "33.0.3"}: "1e59afd40a74c5c0eab0a9fad3f0faf8a674267106e0b19921be9f67081808c2",
	}
	platformToolsOsVersion := [2]string{OS, PLATFORM_TOOLS_VERSION}
	_, err := os.Stat(path.Base(plaformToolsUrlMap[platformToolsOsVersion]))
	if err != nil {
		err = downloadFile(plaformToolsUrlMap[platformToolsOsVersion])
		if err != nil {
			return err
		}
	}
	platformToolsZip = path.Base(plaformToolsUrlMap[platformToolsOsVersion])
	err = verifyZip(platformToolsZip, platformToolsChecksumMap[platformToolsOsVersion])
	if err != nil {
		fmt.Println(platformToolsZip + " checksum verification failed")
		return err
	}
	platformToolsPath := cwd + string(os.PathSeparator) + "platform-tools" + string(os.PathSeparator)
	pathEnvironmentVariable := func() string {
		if OS == "windows" {
			return "Path"
		} else {
			return "PATH"
		}
	}()
	_ = os.Setenv(pathEnvironmentVariable, platformToolsPath+string(os.PathListSeparator)+os.Getenv(pathEnvironmentVariable))
	adbPath := platformToolsPath + "adb"
	fastbootPath := platformToolsPath + "fastboot"
	if OS == "windows" {
		adbPath += ".exe"
		fastbootPath += ".exe"
	}
	adb = exec.Command(adbPath)
	fastboot = exec.Command(fastbootPath)
	// Ensure that no platform tools are running before attempting to overwrite them
	killPlatformTools()
	_, err = extractZip(platformToolsZip, cwd)
	return err
}

func getDevices() map[string]string {
	devices := map[string]string{}
	for _, platformToolCommand := range []exec.Cmd{*adb, *fastboot} {
		platformToolCommand.Args = append(platformToolCommand.Args, "devices")
		output, _ := platformToolCommand.Output()
		lines := strings.Split(string(output), "\n")
		if platformToolCommand.Path == adb.Path {
			lines = lines[1:]
		}
		for i, device := range lines {
			if lines[i] != "" && lines[i] != "\r" {
				serialNumber := strings.Split(device, "\t")[0]
				if platformToolCommand.Path == adb.Path {
					device = getProp("ro.product.device", serialNumber)
				} else if platformToolCommand.Path == fastboot.Path {
					device = getVar("product", serialNumber)
					if device == "sdm845" {
						device = "axolotl"
					}
				}
				fmt.Print("Detected " + device + " " + serialNumber)
				if _, ok := deviceFactoryFolderMap[device]; ok {
					devices[serialNumber] = device
					fmt.Println()
				} else {
					fmt.Println(". " + "No matching factory image found")
				}
			}
		}
	}
	return devices
}

// $ fastboot getvar prop
// prop: value
// Finished. Total time: 0.002s
func getVar(prop string, device string) string {
	platformToolCommand := *fastboot
	platformToolCommand.Args = append(fastboot.Args, "-s", device, "getvar", prop)
	out, err := platformToolCommand.CombinedOutput()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, prop) {
			return strings.Trim(strings.Split(line, " ")[1], "\r")
		}
	}
	return ""
}

// $ fastboot flashing get_unlock_ability
// (bootloader) get_unlock_ability: 0
// OKAY [  0.000s]
// Finished. Total time: 0.000s
func getUnlockAbility(device string) string {
	platformToolCommand := *fastboot
	platformToolCommand.Args = append(fastboot.Args, "-s", device, "flashing", "get_unlock_ability")
	out, err := platformToolCommand.CombinedOutput()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "get_unlock_ability") {
			return strings.Trim(strings.Split(line, " ")[2], "\r")
		}
	}
	return ""
}

// $ fastboot oem device-info
// (bootloader) Verity mode: false
// (bootloader) Device unlocked: true
// (bootloader) Device critical unlocked: true
// (bootloader) Charger screen enabled: false
// OKAY [  0.000s]
// Finished. Total time: 0.000s
func getCriticalUnlocked(device string) string {
	platformToolCommand := *fastboot
	platformToolCommand.Args = append(fastboot.Args, "-s", device, "oem", "device-info")
	out, err := platformToolCommand.CombinedOutput()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Device critical unlocked:") {
			return strings.Trim(strings.Split(line, " ")[4], "\r")
		}
	}
	return ""
}

func getProp(prop string, device string) string {
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "-s", device, "shell", "getprop", prop)
	out, err := platformToolCommand.Output()
	if err != nil {
		return ""
	}
	return strings.Trim(string(out), "[]\n\r")
}

func flashDevices(devices map[string]string) {
	var wg sync.WaitGroup
	for serialNumber, device := range devices {
		wg.Add(1)
		go func(serialNumber, device string) {
			defer wg.Done()
			platformToolCommand := *adb
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "reboot", "bootloader")
			_ = platformToolCommand.Run()
			fmt.Println("Unlocking " + device + " " + serialNumber + " bootloader...")
			warnln("5. Please use the volume and power keys on the device to unlock the bootloader")
			for i := 0; getVar("unlocked", serialNumber) != "yes"; i++ {
				platformToolCommand = *fastboot
				platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "flashing", "unlock")
				_ = platformToolCommand.Start()
				time.Sleep(30 * time.Second)
				if i >= 2 {
					errorln("Failed to unlock "+device+" "+serialNumber+" bootloader", false)
					return
				}
			}
			if device == "FP4" {
				for i := 0; getCriticalUnlocked(serialNumber) != "true"; i++ {
					fmt.Println("Unlocking (critical) " + device + " " + serialNumber + " bootloader...")
					warnln("5.1. Please use the volume and power keys on the device to unlock the bootloader (critical)")
					fmt.Println()
					platformToolCommand = *fastboot
					platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "flashing", "unlock_critical")
					_ = platformToolCommand.Start()
					time.Sleep(30 * time.Second)
					if i >= 2 {
						errorln("Failed to unlock (critical) "+device+" "+serialNumber+" bootloader", false)
						return
					}
				}
			}
			fmt.Println("Flashing " + device + " " + serialNumber + " bootloader...")
			flashAll := exec.Command("." + string(os.PathSeparator) + "flash-all" + func() string {
				if OS == "windows" {
					return ".bat"
				} else {
					return ".sh"
				}
			}())
			flashAll.Dir = deviceFactoryFolderMap[device]
			flashAll.Stderr = os.Stderr
			flashAll.Env = append(flashAll.Environ(), "ANDROID_SERIAL="+serialNumber)
			err := flashAll.Run()
			if err != nil {
				errorln("Failed to flash "+device+" "+serialNumber, false)
				errorln(err.Error(), false)
				return
			}
			fmt.Println("Locking " + device + " " + serialNumber + " bootloader...")
			warnln("6. Please use the volume and power keys on the device to lock the bootloader")
			for i := 0; getVar("unlocked", serialNumber) != "no"; i++ {
				if device == "FP4" && getUnlockAbility(serialNumber) != "1" {
					errorln("Not locking bootloader of "+device+" "+serialNumber, false)
					errorln("fastboot flashing get_unlock_ability returned 0", false)
					errorln("Please visit https://calyxos.org/FP4 for more information.", true)
					return
				}
				platformToolCommand = *fastboot
				platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "flashing", "lock")
				_ = platformToolCommand.Start()
				time.Sleep(30 * time.Second)
				if i >= 2 {
					errorln("Failed to lock "+device+" "+serialNumber+" bootloader", false)
					return
				}
			}
			fmt.Println("Rebooting " + device + " " + serialNumber + "...")
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "reboot")
			_ = platformToolCommand.Start()
		}(serialNumber, device)
	}
	wg.Wait()
	fmt.Println()
	fmt.Println(Blue("Flashing complete"))
}

func killPlatformTools() {
	_, err := os.Stat(adb.Path)
	if err == nil {
		platformToolCommand := *adb
		platformToolCommand.Args = append(platformToolCommand.Args, "kill-server")
		_ = platformToolCommand.Run()
	}
	if OS == "windows" {
		_ = exec.Command("taskkill", "/IM", "fastboot.exe", "/F").Run()
	}
}

func downloadFile(url string) error {
	fmt.Println("Downloading " + url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path.Base(url))
	if err != nil {
		return err
	}
	defer out.Close()

	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	fmt.Println()
	return err
}

func extractZip(src string, destination string) ([]string, error) {
	fmt.Println("Extracting " + src)
	var filenames []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destination, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s is an illegal filepath", fpath)
		}
		filenames = append(filenames, fpath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}
		outFile, err := os.OpenFile(fpath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			f.Mode())
		if err != nil {
			return filenames, err
		}
		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func verifyZip(zipfile, sha256sum string) error {
	fmt.Println("Verifying " + zipfile)
	f, err := os.Open(zipfile)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if sha256sum == sum {
		return nil
	}
	return errors.New("sha256sum mismatch")
}

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	fmt.Printf("\rDownloading... %s downloaded", Bytes(wc.Total))
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanateBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}

func Bytes(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1000, sizes)
}
