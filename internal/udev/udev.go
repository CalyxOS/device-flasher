package udev

import (
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
)

const (
	UdevRules = "# Google\nSUBSYSTEM==\"usb\", ATTR{idVendor}==\"18d1\", GROUP=\"sudo\"\n# Xiaomi\nSUBSYSTEM==\"usb\", ATTR{idVendor}==\"2717\", GROUP=\"sudo\"\n"
	RulesFile = "98-device-flasher.rules"
	RulesPath = "/etc/udev/rules.d2/"
)

func Setup(logger *logrus.Logger) error {
	_, err := os.Stat(RulesPath)
	if os.IsNotExist(err) {
		logger.Debugf("running mkdir %v", RulesPath)
		err = exec.Command("sudo", "mkdir", RulesPath).Run()
		if err != nil {
			return err
		}
		_, err = os.Stat(RulesFile)
		if os.IsNotExist(err) {
			err = ioutil.WriteFile(RulesFile, []byte(UdevRules), 0644)
			return err
		}
		logger.Debugf("running cp %v %v", RulesFile, RulesPath)
		err = exec.Command("sudo", "cp", RulesFile, RulesPath).Run()
		if err != nil {
			return err
		}
		err = exec.Command("sudo", "udevadm", "control", "--reload-rules").Run()
		if err != nil {
			logger.Debugf("udevadm control --reload-rules failed: %v", err)
		}
		err = exec.Command("sudo", "udevadm", "trigger").Run()
		if err != nil {
			logger.Debugf("udevadm trigger failed: %v", err)
		}
	}
	return nil
}
