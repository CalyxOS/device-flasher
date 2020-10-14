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

type ADBFlasher interface {
	GetDeviceIds() ([]string, error)
	GetDeviceCodename(deviceId string) (string, error)
	RebootIntoBootloader(deviceId string) error
	KillServer() error
}

type FastbootFlasher interface {
	GetDeviceIds() ([]string, error)
	GetDeviceCodename(deviceId string) (string, error)
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
	defer f.adb.KillServer()

	logger := f.logger.WithFields(logrus.Fields{
		"deviceId":       device.ID,
		"deviceCodename": device.Codename,
	})

	logger.Info("validating factory image is for device")
	err := f.factoryImage.Validate(device.Codename)
	if err != nil {
		return err
	}

	logger.Info("reboot into bootloader")
	err = f.adb.RebootIntoBootloader(device.ID)
	if err != nil {
		f.logger.Debugf("ignoring adb reboot error and will attempt fastboot access: %v", err)
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
		return err
	}
	logger.Info("finished flashing device")

	return nil
}

func (f *Flash) DiscoverDevices() ([]*Device, error) {
	deviceIds, err := f.getDeviceIds()
	if err != nil {
		return nil, err
	}

	devices, err := f.generateDevices(deviceIds)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

func (f *Flash) getDeviceIds() ([]string, error) {
	f.logger.Debug("running adb get devices")
	deviceIds, err := f.adb.GetDeviceIds()
	if err != nil {
		f.logger.Infof("unable to get adb devices: %v", err)
	}
	if len(deviceIds) == 0 {
		f.logger.Debug("running fastboot get devices")
		deviceIds, err = f.fastboot.GetDeviceIds()
		if err != nil {
			f.logger.Infof("unable to get fastboot devices: %v", err)
		}
		if len(deviceIds) == 0 {
			return nil, ErrNoDevicesFound
		}
	}
	return deviceIds, nil
}

func (f *Flash) getDeviceCodename(deviceId string) (string, error) {
	f.logger.Debugf("getting code name for device %v", deviceId)
	deviceCodename, err := f.adb.GetDeviceCodename(deviceId)
	if err != nil {
		f.logger.Debugf("unable to get code name through adb for %v: %v", deviceId, err)
		deviceCodename, err = f.fastboot.GetDeviceCodename(deviceId)
		if err != nil {
			f.logger.Debugf("unable to get code name through fastboot for %v: %v", deviceId, err)
			return "", fmt.Errorf("cannot determine device model for %v: %w", deviceId, err)
		}
	}
	return deviceCodename, nil
}

func (f *Flash) generateDevices(deviceIds []string) ([]*Device, error) {
	var devices []*Device
	for _, deviceId := range deviceIds {
		deviceCodename, err := f.getDeviceCodename(deviceId)
		if err != nil {
			return nil, err
		}
		devices = append(devices, &Device{
			ID:       deviceId,
			Codename: deviceCodename,
		})
	}
	return devices, nil
}
