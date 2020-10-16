package devicediscovery

import (
	"github.com/sirupsen/logrus"
)

var CustomFlashHooks = map[Codename]*FlashHooks{
	Codename("jasmine"): &FlashHooks{
		HookPreUnlock: additionalLockUnlockStep,
		HookPreLock: additionalLockUnlockStep,
	},
	Codename("walleye"): &FlashHooks{
		HookPreUnlock: additionalLockUnlockStep,
		HookPreLock: additionalLockUnlockStep,
	},
}

type Codename string
type Hook func(logger *logrus.Logger) error

func additionalLockUnlockStep(logger *logrus.Logger) error {
	logger.Info("-> Once device boots, disconnect its cable and power it off")
	logger.Info("-> Then, press volume down + power to boot it into fastboot mode, and connect the cable again.")
	logger.Info("The installation will resume automatically")
	return nil
}

type FlashHooks struct {
	HookPreUnlock Hook
	HookPreLock   Hook
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




