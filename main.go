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
	toolsVersionPtr := flag.String("tools-version", string(platformtools.Version_30_0_4),
		fmt.Sprintf("platform tools version. supported: %v", platformtools.SupportedVersions))
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
		err := udev.Setup(logger, udev.DefaultUDevRules)
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
	tmpToolExtractDir, err := ioutil.TempDir("", "device-flasher-extracted-platformtools")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpToolExtractDir)

	logger.Debug("creating cache dir for downloaded platform tools zips")
	toolZipCacheDir := fmt.Sprintf("%v%v%v%v", os.TempDir(), string(os.PathSeparator), "platform-tools-", toolsVersion)
	err = os.MkdirAll(toolZipCacheDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to setup tools zip cache dir %v: %w", toolZipCacheDir, err)
	}
	platformTools, err := platformtools.New(&platformtools.Config{
		CacheDir:             toolZipCacheDir,
		HttpClient:           &http.Client{Timeout: time.Second * 60},
		HostOS:               hostOS,
		ToolsVersion:         toolsVersion,
		DestinationDirectory: tmpToolExtractDir,
		Logger:               logger,
	})
	if err != nil {
		return err
	}

	logger.Info("setting up adb")
	adbTool, err := adb.New(platformTools.Path(), hostOS)
	if err != nil {
		logger.Fatal(err)
	}
	err = adbTool.KillServer()
	if err != nil {
		logger.Debug(err)
	}
	err = adbTool.StartServer()
	if err != nil {
		logger.Warn(err)
	}
	defer adbTool.KillServer()

	logger.Info("setting up fastboot")
	fastbootTool, err := fastboot.New(platformTools.Path(), hostOS)
	if err != nil {
		return err
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
	devicesMap, err := flashTool.DiscoverDevices()
	if err != nil {
		return err
	}

	logger.Info("Discovered the following devices:")
	for _, device := range devicesMap {
		logger.Infof(" id=%v codename=%v (%v)", device.ID, device.Codename, device.DiscoveryTool)
	}
	logger.Warn("Press enter to start flashing process")
	_, _ = fmt.Scanln()

	// keep serial for the time being until everything is working
	for _, device := range devicesMap {
		deviceLogger := logger.WithFields(logrus.Fields{
			"deviceId":       device.ID,
			"deviceCodename": device.Codename,
		})
		deviceLogger.Infof("starting to flash device")
		err = flashTool.Flash(device)
		if err != nil {
			deviceLogger.Error(err)
			return err
		}
		deviceLogger.Infof("finished flashing device")
	}
	return nil
}
