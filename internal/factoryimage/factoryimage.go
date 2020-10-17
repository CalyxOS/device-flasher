package factoryimage

import (
	"errors"
	"fmt"
	"github.com/mholt/archiver/v3"
	"github.com/sirupsen/logrus"
	"gitlab.com/calyxos/device-flasher/internal/device"
	"gitlab.com/calyxos/device-flasher/internal/platformtools"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrorFailedToFlash = errors.New("failed to flash device")
	ErrorValidation    = errors.New("failed to validate image for device")
)

type Config struct {
	HostOS           string
	Name             string
	ImagePath        string
	WorkingDirectory string
	Logger           *logrus.Logger
}

type FactoryImage struct {
	hostOS             string
	extractedDirectory string
	name               string
	flashAllScript     string
	imagePath          string
	workingDirectory   string
	logger             *logrus.Logger
}

func New(config *Config) (*FactoryImage, error) {
	factoryImage := &FactoryImage{
		hostOS:           config.HostOS,
		name:             config.Name,
		workingDirectory: config.WorkingDirectory,
		imagePath:        config.ImagePath,
		logger:           config.Logger,
	}
	err := factoryImage.setup()
	if err != nil {
		return nil, err
	}
	return factoryImage, nil
}

func (f *FactoryImage) FlashAll(platformToolsPath platformtools.PlatformToolsPath) error {
	pathEnvironmentVariable := "PATH"
	if f.hostOS == "windows" {
		pathEnvironmentVariable = "Path"
	}

	path := os.Getenv(pathEnvironmentVariable)
	newPath := string(platformToolsPath) + string(os.PathListSeparator) + path
	f.logger.WithField("newPath", newPath).Info("adding platform tools to PATH")
	err := os.Setenv(pathEnvironmentVariable, newPath)
	if err != nil {
		return err
	}

	flashAll := fmt.Sprintf("./%v", f.flashAllScript)
	f.logger.WithFields(logrus.Fields{
		"flashAll": flashAll,
	}).Debug("running flash all script on device")
	flashCmd := exec.Command(flashAll)
	flashCmd.Dir = f.extractedDirectory
	flashCmd.Stdout = os.Stdout
	flashCmd.Stderr = os.Stdout
	err = flashCmd.Run()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrorFailedToFlash, err)
	}
	return nil
}

func (f *FactoryImage) Validate(deviceCodename device.Codename) error {
	f.logger.WithFields(logrus.Fields{
		"deviceCodename": deviceCodename,
	}).Info("running factory image validation")
	if _, err := os.Stat(f.imagePath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %v", ErrorValidation, err)
	}
	if !strings.Contains(f.imagePath, strings.ToLower(string(deviceCodename))) {
		return fmt.Errorf("%w: image filename should contain device codename %v", ErrorValidation, deviceCodename)
	}
	if !strings.HasSuffix(f.imagePath, ".zip") {
		return fmt.Errorf("%w: image filename should end in .zip", ErrorValidation)
	}
	if !strings.Contains(f.imagePath, "factory") {
		return fmt.Errorf("%w: image filename should contain 'factory'", ErrorValidation)
	}
	return nil
}

func (f *FactoryImage) setup() error {
	err := f.extract()
	if err != nil {
		return err
	}

	err = f.postExtractValidations()
	if err != nil {
		return err
	}

	return nil
}

func (f *FactoryImage) extract() error {
	f.logger.WithFields(logrus.Fields{
		"name":      f.name,
		"imagePath": f.imagePath,
	}).Info("extracting factory image")
	err := archiver.Unarchive(f.imagePath, f.workingDirectory)
	if err != nil {
		return err
	}
	return nil
}

func (f *FactoryImage) postExtractValidations() error {
	f.flashAllScript = "flash-all.sh"
	if f.hostOS == "windows" {
		f.flashAllScript = "flash-all.bat"
	}

	// TODO this can probably be simplified
	files, err := ioutil.ReadDir(f.workingDirectory)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			_, err := os.Stat(f.workingDirectory + file.Name() + string(os.PathSeparator) + f.flashAllScript)
			if err != nil {
				f.extractedDirectory = f.workingDirectory + string(os.PathSeparator) + file.Name()
			}
		}
	}
	if f.extractedDirectory == "" {
		return fmt.Errorf("unable to find %v in directory %v", f.flashAllScript, f.workingDirectory)
	}

	f.logger.WithFields(logrus.Fields{
		"flashAllScript":     f.flashAllScript,
		"extractedDirectory": f.extractedDirectory,
	}).Debug("validated extracted factory image")

	return nil
}
