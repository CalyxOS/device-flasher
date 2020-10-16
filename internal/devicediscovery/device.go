package devicediscovery

import (
	"github.com/sirupsen/logrus"
)

var CustomFlashHooks = map[Codename]*FlashHooks{
	Codename("crosshatch"): &FlashHooks{
		RenameCodename: Codename("jasmin_sprout"),
		LoggingHookPreUnlock: additionalLockUnlockStep,
		LoggingHookPreLock: additionalLockUnlockStep,
	},
	Codename("walleye"): &FlashHooks{
		LoggingHookPreUnlock: additionalLockUnlockStep,
		LoggingHookPreLock: additionalLockUnlockStep,
	},
}

type Codename string
type LoggingHook func(logger *logrus.Logger) error

func additionalLockUnlockStep(logger *logrus.Logger) error {
	logger.Info("-> Once device boots, disconnect its cable and power it off")
	logger.Info("-> Then, press volume down + power to boot it into fastboot mode, and connect the cable again.")
	logger.Info("The installation will resume automatically")
	return nil
}

type FlashHooks struct {
	RenameCodename       Codename
	LoggingHookPreUnlock LoggingHook
	LoggingHookPreLock   LoggingHook
}

type ToolName string
const (
	ADB      ToolName = "adb"
	Fastboot ToolName = "fastboot"
)

type Device struct {
	ID            string
	Codename      Codename
	DiscoveryTool ToolName
	FlashHooks    *FlashHooks
}




