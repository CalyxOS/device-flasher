//go:generate mockgen -destination=mocks/mocks.go -package=mocks . FactoryImageFlasher,PlatformToolsFlasher,ADBFlasher,FastbootFlasher
package flash

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/calyxos/device-flasher/internal/devicediscovery"
	"gitlab.com/calyxos/device-flasher/internal/platformtools"
	"gitlab.com/calyxos/device-flasher/internal/platformtools/fastboot"
)

type FactoryImageFlasher interface {
	Validate(codename devicediscovery.Codename) error
	FlashAll(path platformtools.PlatformToolsPath) error
}

type PlatformToolsFlasher interface {
	Path() platformtools.PlatformToolsPath
}

type ADBFlasher interface {
	RebootIntoBootloader(deviceId string) error
	KillServer() error
}

type FastbootFlasher interface {
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

func (f *Flash) Flash(device *devicediscovery.Device) error {
	logger := f.logger.WithFields(logrus.Fields{
		"deviceId":       device.ID,
		"deviceCodename": device.Codename,
	})

	logger.Info("validating factory image is for device")
	err := f.factoryImage.Validate(device.Codename)
	if err != nil {
		return err
	}

	if device.DiscoveryTool == devicediscovery.ADB {
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
		logger.Info("-> Please use the volume and power keys on the device to unlock the bootloader")
		if device.FlashHooks != nil {
			err := device.FlashHooks.LoggingHookPreUnlock(logger.Logger)
			if err != nil {
				logger.Warnf("pre unlock hook failed: %v", err)
			}
		}
		err = f.fastboot.SetBootloaderLockStatus(device.ID, fastboot.Unlocked)
		if err != nil {
			return err
		}
	}
	logger.Infof("bootloader is unlocked")

	logger.Info("running flash all script")
	err = f.factoryImage.FlashAll(f.platformTools.Path())
	if err != nil {
		return err
	}
	logger.Info("finished running flash all script")

	logger.Info("re-locking bootloader")
	logger.Info("-> Please use the volume and power keys on the device to unlock the bootloader")
	if device.FlashHooks != nil {
		err = device.FlashHooks.LoggingHookPreLock(logger.Logger)
		if err != nil {
			logger.Warnf("pre lock hook failed: %v", err)
		}
	}
	err = f.fastboot.SetBootloaderLockStatus(device.ID, fastboot.Locked)
	if err != nil {
		return err
	}
	logger.Info("bootloader is now locked")

	logger.Info("rebooting device")
	err = f.fastboot.Reboot(device.ID)
	if err != nil {
		logger.Warnf("failed to reboot device: %v. may need to manually reboot", err)
	}
	logger.Info("-> Disable OEM unlocking from Developer Options after setting up your device")

	return nil
}