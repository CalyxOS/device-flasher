package main

import (
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"gitlab.com/calyxos/device-flasher/internal/factoryimage"
	"gitlab.com/calyxos/device-flasher/internal/flash"
	"gitlab.com/calyxos/device-flasher/internal/platformtools"
	"gitlab.com/calyxos/device-flasher/internal/platformtools/adb"
	"gitlab.com/calyxos/device-flasher/internal/platformtools/fastboot"
	"gitlab.com/calyxos/device-flasher/internal/udev"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"
)

func main() {
	// TODO: changed to latest as darwin r30.0.2 did not exist
	// also might be better to just hardcode version in platformtools
	toolsVersionPtr := flag.String("tools-version", "latest", "platform tools version")
	namePtr := flag.String("name", "CalyxOS", "os name")
	imagePtr := flag.String("image", "", "factory image to flash")
	debugPtr := flag.Bool("debug", false, "debug logging")
	flag.Parse()

	var logger = logrus.New()
	if *debugPtr {
		logger.SetLevel(logrus.DebugLevel)
	}

	if *imagePtr == "" {
		logger.Fatal("must specify factory image")
	}

	err := execute(*namePtr, *imagePtr, *toolsVersionPtr, runtime.GOOS, logger)
	if err != nil {
		logger.Fatal(err)
	}
}

func execute(name, image, toolsVersion, hostOS string, logger *logrus.Logger) error {
	// setup udev if running linux
	if hostOS == "linux" {
		err := udev.Setup(logger)
		if err != nil {
			return err
		}
	}

	// factory image setup
	logger.Debug("creating temporary directory for extracting factory image")
	tmpFactoryDir, err := ioutil.TempDir("", "device-flasher-factory")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpFactoryDir)
	factoryImage, err := factoryimage.New(&factoryimage.Config{
		HostOS:           hostOS,
		Name:             name,
		ImagePath:        image,
		WorkingDirectory: tmpFactoryDir,
		Logger:           logger,
	})
	if err != nil {
		return err
	}

	// platform tools setup
	logger.Debug("creating temporary directory for platform tools")
	tmpToolDir, err := ioutil.TempDir("", "device-flasher-platformtools")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpToolDir)
	platformTools, err := platformtools.New(&platformtools.Config{
		HttpClient:           &http.Client{Timeout: time.Second * 60},
		HostOS:               hostOS,
		ToolsVersion:         toolsVersion,
		DestinationDirectory: tmpToolDir,
		Logger:               logger,
	})
	if err != nil {
		logger.Fatal(err)
	}

	adbTool, err := adb.New(platformTools.Path(), hostOS)
	if err != nil {
		logger.Fatal(err)
	}

	fastbootTool, err := fastboot.New(platformTools.Path(), hostOS)
	if err != nil {
		logger.Fatal(err)
	}

	flashTool := flash.New(&flash.Config{
		HostOS:        hostOS,
		FactoryImage:  factoryImage,
		PlatformTools: platformTools,
		ADB:           adbTool,
		Fastboot:      fastbootTool,
		Logger:        logger,
	})

	logger.Info("Connect to a wifi network and ensure that no SIM cards are installed")
	logger.Info("Enable Developer Options on device (Settings -> About Phone -> tap \"Build number\" 7 times)")
	logger.Info("Enable USB debugging on device (Settings -> System -> Advanced -> Developer Options) and allow the computer to debug (hit \"OK\" on the popup when USB is connected)")
	logger.Info("Enable OEM Unlocking (in the same Developer Options menu)")
	logger.Warn("When done, press enter to continue")
	_, _ = fmt.Scanln()
	devices, err := flashTool.DiscoverDevices()
	if err != nil {
		return err
	}

	logger.Info("detected the following devices:")
	for _, device := range devices {
		logger.Infof("  id:%v codename:%v discovery:%v", device.ID, device.Codename, device.DiscoveryTool)
	}

	logger.Info("running flash devices")
	for _, device := range devices {
		err = flashTool.Flash(device)
		if err != nil {
			return err
		}
	}
	return nil
}
