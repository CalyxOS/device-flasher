//go:generate mockgen -destination=mocks/mocks.go -package=mocks . FactoryImageFlasher,PlatformToolsFlasher,ADBFlasher,FastbootFlasher
package flash

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gitlab.com/calyxos/device-flasher/internal/platformtools"
	"gitlab.com/calyxos/device-flasher/internal/platformtools/fastboot"
)

var (
	ErrNoDevicesFound = errors.New("no devices detected with adb or fastboot")
)

type FactoryImageFlasher interface {
	Validate(deviceCodename string) error
	FlashAll(path platformtools.PlatformToolsPath) error
}

type PlatformToolsFlasher interface {
	Path() platformtools.PlatformToolsPath
}

type DeviceDiscoverer interface {
	GetDeviceIds() ([]string, error)
	GetDeviceCodename(deviceId string) (string, error)
}

type ADBFlasher interface {
	DeviceDiscoverer
	RebootIntoBootloader(deviceId string) error
	KillServer() error
}

type FastbootFlasher interface {
	DeviceDiscoverer
	SetBootloaderLockStatus(deviceId string, wanted fastboot.FastbootLockStatus) error
	GetBootloaderLockStatus(deviceId string) (fastboot.FastbootLockStatus, error)
	Reboot(deviceId string) error
}

type Config struct {
	HostOS        string
	FactoryImage  FactoryImageFlasher
	PlatformTools PlatformToolsFlasher
	ADB           ADBFlasher
	Fastboot      FastbootFlasher
	Logger        *logrus.Logger
}

type Flash struct {
	hostOS        string
	factoryImage  FactoryImageFlasher
	platformTools PlatformToolsFlasher
	adb           ADBFlasher
	fastboot      FastbootFlasher
	logger        *logrus.Logger
}

func New(config *Config) *Flash {
	return &Flash{
		hostOS:        config.HostOS,
		factoryImage:  config.FactoryImage,
		platformTools: config.PlatformTools,
		adb:           config.ADB,
		fastboot:      config.Fastboot,
		logger:        config.Logger,
	}
}

func (f *Flash) Flash(device *Device) error {
	logger := f.logger.WithFields(logrus.Fields{
		"deviceId":       device.ID,
		"deviceCodename": device.Codename,
	})

	logger.Info("validating factory image is for device")
	err := f.factoryImage.Validate(device.Codename)
	if err != nil {
		return err
	}

	if device.DiscoveryType == ADBDiscovered {
		logger.Info("reboot into bootloader")
		err = f.adb.RebootIntoBootloader(device.ID)
		if err != nil {
			f.logger.Debugf("ignoring adb reboot error and will attempt fastboot access: %v", err)
		}
	}

	logger.Info("checking bootloader status")
	lockStatus, err := f.fastboot.GetBootloaderLockStatus(device.ID)
	if err != nil {
		return err
	}
	if lockStatus != fastboot.Unlocked {
		logger.Info("unlocking bootloader")
		err := f.fastboot.SetBootloaderLockStatus(device.ID, fastboot.Unlocked)
		if err != nil {
			return err
		}
	}
	logger.Infof("bootloader is unlocked")

	logger.Info("running flash all script")
	err = f.factoryImage.FlashAll(f.platformTools.Path())
	if err != nil {
		return fmt.Errorf("failed to flash device %v: %w", device.ID, err)
	}
	logger.Info("finished running flash all script")

	logger.Info("re-locking bootloader")
	err = f.fastboot.SetBootloaderLockStatus(device.ID, fastboot.Locked)
	if err != nil {
		return err
	}
	logger.Info("bootloader is locked")

	logger.Info("rebooting device")
	err = f.fastboot.Reboot(device.ID)
	if err != nil {
		logger.Warnf("failed to reboot device: %v. may need to manually reboot", err)
	}
	logger.Info("finished flashing device")

	return nil
}

func (f *Flash) DiscoverDevices() (map[string]*Device, error) {
	devices := map[string]*Device{}
	f.logger.Debug("running adb get devices")
	adbDeviceIds, err := f.adb.GetDeviceIds()
	if err != nil {
		f.logger.Infof("unable to get adb devices: %v", err)
	} else {
		devices, err = f.generateDevices(adbDeviceIds, f.adb, ADBDiscovered)
		if err != nil {
			return nil, err
		}
	}

	fastbootDevices := map[string]*Device{}
	f.logger.Debug("running fastboot get devices")
	fastbootDeviceIds, err := f.fastboot.GetDeviceIds()
	if err != nil {
		f.logger.Infof("unable to get fastboot devices: %v", err)
	} else {
		fastbootDevices, err = f.generateDevices(fastbootDeviceIds, f.fastboot, FastbootDiscovered)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range fastbootDevices {
		devices[k] = v
	}

	if len(devices) == 0 {
		return nil, ErrNoDevicesFound
	}

	return devices, nil
}

func (f *Flash) generateDevices(deviceIds []string, tool DeviceDiscoverer, discoverType DiscoveryType) (map[string]*Device, error) {
	devices := map[string]*Device{}
	for _, deviceId := range deviceIds {
		f.logger.Debugf("getting code name for device %v", deviceId)
		deviceCodename, err := tool.GetDeviceCodename(deviceId)
		if err != nil {
			f.logger.Warnf("skipping device %v as getting code name failed: %v", deviceId, err)
			continue
		}
		if _, ok := devices[deviceId]; ok {
			f.logger.Warnf("skipping duplicate device %v", deviceId)
			continue
		}
		devices[deviceId] = &Device{
			ID: deviceId,
			Codename: deviceCodename,
			DiscoveryType: discoverType,
		}
	}
	return devices, nil
}