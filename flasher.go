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
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var input string

var executable, _ = os.Executable()
var cwd = filepath.Dir(executable)

var adb *exec.Cmd
var fastboot *exec.Cmd

var platformToolsVersion = "30.0.4"
var platformToolsZip string

var factoryFiles map[string]FactoryImage

type FactoryImage struct {
	avb        string
	bootloader string
	radio      string
	image      string
}

const OS = runtime.GOOS

const (
	UDEV_RULES = "# Google\nSUBSYSTEM==\"usb\", ATTR{idVendor}==\"18d1\", GROUP=\"sudo\"\n# Xiaomi\nSUBSYSTEM==\"usb\", ATTR{idVendor}==\"2717\", GROUP=\"sudo\"\n"
	RULES_FILE = "98-device-flasher.rules"
	RULES_PATH = "/etc/udev/rules.d2/"
)

var (
	Warn  = Yellow
	Error = Red
)

var (
	Red    = Color("\033[1;31m%s\033[0m")
	Yellow = Color("\033[1;33m%s\033[0m")
)

func Color(color string) func(...interface{}) string {
	return func(args ...interface{}) string {
		return fmt.Sprintf(color,
			fmt.Sprint(args...))
	}
}

func fatalln(err error) {
	log, _ := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	_, _ = fmt.Fprintln(log, err.Error())
	_, _ = fmt.Fprintln(os.Stderr, Error(err.Error()))
	log.Close()
	cleanup()
	os.Exit(1)
}

func errorln(err string) {
	log, _ := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	defer log.Close()
	_, _ = fmt.Fprintln(log, err)
	_, _ = fmt.Fprintln(os.Stderr, Error(err))
}

func cleanup() {
	_, err := os.Stat(RULES_PATH + RULES_FILE)
	if !os.IsNotExist(err) {
		_ = exec.Command("sudo", "rm", RULES_PATH+RULES_FILE).Run()
	}
}

func main() {
	_ = os.Remove("error.log")
	factoryFiles = getFactoryFiles()
	if len(factoryFiles) < 1 {
		fatalln(errors.New("Cannot continue without a device factory image. Exiting..."))
	}
	platformToolsZip = "platform-tools_r" + platformToolsVersion + "-" + OS + ".zip"
	err := getPlatformTools()
	if err != nil {
		errorln("Cannot continue without Android platform tools. Exiting...")
		fatalln(err)
	}
	if OS == "linux" {
		checkUdevRules()
	}
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "start-server")
	err = platformToolCommand.Run()
	if err != nil {
		errorln("Cannot start ADB server")
		fatalln(err)
	}
	fmt.Println("Do the following for each device:")
	fmt.Println("Connect to a wifi network and ensure that no SIM cards are installed")
	fmt.Println("Enable Developer Options on device (Settings -> About Phone -> tap \"Build number\" 7 times)")
	fmt.Println("Enable USB debugging on device (Settings -> System -> Advanced -> Developer Options) and allow the computer to debug (hit \"OK\" on the popup when USB is connected)")
	fmt.Println("Enable OEM Unlocking (in the same Developer Options menu)")
	fmt.Print("When done, press enter to continue")
	_, _ = fmt.Scanln(&input)
	devices := getDevices()
	if len(devices) == 0 {
		fatalln(errors.New("No device connected. Exiting..."))
	}
	fmt.Print("Detected " + strconv.Itoa(len(devices)) + " devices: ")
	fmt.Println(reflect.ValueOf(devices).MapKeys())
	flashDevices(devices)
	defer cleanup()
}

func getPlatformTools() error {
	platformToolsPath := cwd + string(os.PathSeparator) + "platform-tools" + string(os.PathSeparator)
	adbPath := platformToolsPath + "adb"
	fastbootPath := platformToolsPath + "fastboot"
	if OS == "windows" {
		adbPath += ".exe"
		fastbootPath += ".exe"
	}
	adb = exec.Command(adbPath)
	fastboot = exec.Command(fastbootPath)
	platformToolsChecksum := map[string]string{
		"platform-tools_r29.0.6-darwin.zip":  "7555e8e24958cae4cfd197135950359b9fe8373d4862a03677f089d215119a3a",
		"platform-tools_r29.0.6-linux.zip":   "cc9e9d0224d1a917bad71fe12d209dfffe9ce43395e048ab2f07dcfc21101d44",
		"platform-tools_r29.0.6-windows.zip": "247210e3c12453545f8e1f76e55de3559c03f2d785487b2e4ac00fe9698a039c",
		"platform-tools_r30.0.4-darwin.zip":  "e0db2bdc784c41847f854d6608e91597ebc3cef66686f647125f5a046068a890",
		"platform-tools_r30.0.4-linux.zip":   "5be24ed897c7e061ba800bfa7b9ebb4b0f8958cc062f4b2202701e02f2725891",
		"platform-tools_r30.0.4-windows.zip": "413182fff6c5957911e231b9e97e6be4fc6a539035e3dfb580b5c54bd5950fee",
	}
	_, err := os.Stat(platformToolsZip)
	if err == nil {
		killPlatformTools()
		err = verifyZip(platformToolsZip, platformToolsChecksum[platformToolsZip])
	}
	if err != nil {
		url := "https://dl.google.com/android/repository/"
		if OS == "darwin" {
			url += "fbad467867e935dce68a0296b00e6d1e76f15b15."
		}
		url += platformToolsZip
		err = downloadFile(url)
		if err != nil {
			return err
		}
		err = verifyZip(platformToolsZip, platformToolsChecksum[platformToolsZip])
		if err != nil {
			fmt.Println(platformToolsZip + " checksum verification failed")
			return err
		}
	}
	_, err = extractZip(platformToolsZip, cwd)
	return err
}

func checkUdevRules() {
	_, err := os.Stat(RULES_PATH)
	if os.IsNotExist(err) {
		err = exec.Command("sudo", "mkdir", RULES_PATH).Run()
		if err != nil {
			errorln("Cannot continue without udev rules. Exiting...")
			fatalln(err)
		}
		_, err = os.Stat(RULES_FILE)
		if os.IsNotExist(err) {
			err = ioutil.WriteFile(RULES_FILE, []byte(UDEV_RULES), 0644)
			if err != nil {
				errorln("Cannot continue without udev rules. Exiting...")
				fatalln(err)
			}
		}
		err = exec.Command("sudo", "cp", RULES_FILE, RULES_PATH).Run()
		if err != nil {
			errorln("Cannot continue without udev rules. Exiting...")
			fatalln(err)
		}
		_ = exec.Command("sudo", "udevadm", "control", "--reload-rules").Run()
		_ = exec.Command("sudo", "udevadm", "trigger").Run()
	}
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
				}
				if _, ok := factoryFiles[device]; ok {
					devices[serialNumber] = device
				}
			}
		}
	}
	return devices
}

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

func getProp(prop string, device string) string {
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "-s", device, "shell", "getprop", prop)
	out, err := platformToolCommand.Output()
	if err != nil {
		return ""
	}
	return strings.Trim(string(out), "[]\n\r")
}

func getFactoryFiles() map[string]FactoryImage {
	files, err := ioutil.ReadDir(cwd)
	if err != nil {
		fatalln(err)
	}
	factoryFiles := map[string]FactoryImage{}
	var wg sync.WaitGroup
	for _, file := range files {
		file := file.Name()
		if strings.Contains(file, "factory") && strings.HasSuffix(file, ".zip") {
			if strings.HasPrefix(file, "jasmine") {
				platformToolsVersion = "29.0.6"
			}
			wg.Add(1)
			go func(file string) {
				defer wg.Done()
				extracted, err := extractZip(path.Base(file), cwd)
				if err != nil {
					errorln("Cannot continue without the device factory image. Exiting...")
					fatalln(err)
				}
				factoryImage := FactoryImage{
					avb:        "",
					bootloader: "",
					radio:      "",
					image:      "",
				}
				for _, file := range extracted {
					if strings.Contains(file, "avb") && strings.HasSuffix(file, ".bin") {
						factoryImage.avb = file
					} else if strings.Contains(file, "bootloader") {
						factoryImage.bootloader = file
					} else if strings.Contains(file, "radio") {
						factoryImage.radio = file
					} else if strings.Contains(file, "image") {
						factoryImage.image = file
					}
				}
				factoryFiles[strings.Split(file, "-")[0]] = factoryImage
			}(file)
		}
	}
	wg.Wait()
	return factoryFiles
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
			fmt.Println("Unlocking device " + serialNumber + " bootloader...")
			fmt.Println("Please use the volume and power keys on the device to confirm.")
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "flashing", "unlock")
			_ = platformToolCommand.Run()
			if getVar("unlocked", serialNumber) != "yes" {
				errorln("Failed to unlock device " + serialNumber + " bootloader")
				return
			}
			platformToolCommand = *fastboot
			fmt.Println("Flashing device " + serialNumber + "...")
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "--slot", "all", "flash", "bootloader", factoryFiles[device].bootloader)
			platformToolCommand.Stderr = os.Stderr
			err := platformToolCommand.Run()
			if err != nil {
				errorln("Failed to flash stock bootloader on device " + serialNumber)
				return
			}
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "reboot-bootloader")
			_ = platformToolCommand.Run()
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "--slot", "all", "flash", "radio", factoryFiles[device].radio)
			platformToolCommand.Stderr = os.Stderr
			err = platformToolCommand.Run()
			if err != nil {
				errorln("Failed to flash stock radio on device " + serialNumber)
				return
			}
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "reboot-bootloader")
			_ = platformToolCommand.Run()
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "--skip-reboot", "update", factoryFiles[device].image)
			platformToolCommand.Stderr = os.Stderr
			err = platformToolCommand.Run()
			if err != nil {
				errorln("Failed to flash device " + serialNumber)
				return
			}
			fmt.Println("Wiping userdata for device " + serialNumber + "...")
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "-w", "reboot-bootloader")
			err = platformToolCommand.Run()
			if err != nil {
				errorln("Failed to wipe userdata for device " + serialNumber)
				return
			}
			if factoryFiles[device].avb != "" {
				fmt.Println("Locking device " + serialNumber + " bootloader...")
				platformToolCommand := *fastboot
				platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "erase", "avb_custom_key")
				err := platformToolCommand.Run()
				if err != nil {
					errorln("Failed to erase avb_custom_key for device " + serialNumber)
					return
				}
				platformToolCommand = *fastboot
				platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "flash", "avb_custom_key", factoryFiles[device].avb)
				err = platformToolCommand.Run()
				if err != nil {
					errorln("Failed to flash avb_custom_key for device " + serialNumber)
					return
				}
				fmt.Println("Please use the volume and power keys on the device to confirm.")
				platformToolCommand = *fastboot
				platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "flashing", "lock")
				_ = platformToolCommand.Run()
				if getVar("unlocked", serialNumber) != "no" {
					errorln("Failed to lock device " + serialNumber + " bootloader")
					return
				}
			}
			fmt.Println("Rebooting " + serialNumber + "...")
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", serialNumber, "reboot")
			_ = platformToolCommand.Start()
		}(serialNumber, device)
	}
	wg.Wait()
	fmt.Println("Bulk flashing complete")
}

func killPlatformTools() {
	platformToolCommand := *adb
	platformToolCommand.Args = append(platformToolCommand.Args, "kill-server")
	err := platformToolCommand.Run()
	if err != nil {
		errorln(err.Error())
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
