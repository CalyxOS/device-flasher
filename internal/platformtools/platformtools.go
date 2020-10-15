package platformtools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/mholt/archiver/v3"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
)

const (
	DefaultBaseURI                = "https://dl.google.com/android/repository"
	PlatformToolsFilenameTemplate = "platform-tools-%v-%v.zip"
)

const (
	LinuxSha256   = "f7306a7c66d8149c4430aff270d6ed644c720ea29ef799dc613d3dc537485c6e"
	DarwinSha256  = "ab9dbab873fff677deb2cfd95ea60b9295ebd53b58ec8533e9e1110b2451e540"
	WindowsSha256 = "265dd7b55f58dff1a5ad5073a92f4a5308bd070b72bd8b0d604674add6db8a41"
)

type PlatformToolsPath string

type ToolName string

const (
	ADB ToolName = "adb"
	Fastboot ToolName = "fastboot"
)

type Config struct {
	BaseURI              string
	HttpClient           *http.Client
	HostOS               string
	ToolsVersion         string
	DestinationDirectory string
	Logger               *logrus.Logger
}

type PlatformTools struct {
	httpClient       *http.Client
	os               string
	downloadURI      string
	sha256           string
	workingDirectory string
	zipFile          string
	path             string
	logger           *logrus.Logger
}

func New(config *Config) (*PlatformTools, error) {
	platformToolsFilename := fmt.Sprintf(PlatformToolsFilenameTemplate, config.ToolsVersion, config.HostOS)
	downloadURI := fmt.Sprintf("%v/%v", DefaultBaseURI, platformToolsFilename)
	workingDirectory := config.DestinationDirectory
	zipFile := fmt.Sprintf("%v/%v", workingDirectory, "platform-tools.zip")
	path := fmt.Sprintf("%v/platform-tools", workingDirectory)

	var sha256 string
	switch config.HostOS {
	case "linux":
		sha256 = LinuxSha256
	case "darwin":
		sha256 = DarwinSha256
	case "windows":
		sha256 = WindowsSha256
	}

	platformTools := &PlatformTools{
		path:             path,
		httpClient:       config.HttpClient,
		downloadURI:      downloadURI,
		sha256:           sha256,
		workingDirectory: workingDirectory,
		zipFile:          zipFile,
		logger:           config.Logger,
	}

	err := platformTools.initialize()
	if err != nil {
		return nil, err
	}

	return platformTools, nil
}

func (p *PlatformTools) initialize() error {
	logger := p.logger.WithFields(logrus.Fields{
		"downloadURI": p.downloadURI,
		"sha256":      p.sha256,
		"zipFile":     p.zipFile,
	})

	logger.Debug("starting tools download")
	err := p.download()
	if err != nil {
		return err
	}

	logger.Debug("starting tools extract")
	err = p.extract()
	if err != nil {
		return err
	}

	// TODO: add back verify
	//logger.Debug("starting tools verify")
	//err = p.verify()
	//if err != nil {
	//	return err
	//}

	return nil
}

func (p *PlatformTools) Path() PlatformToolsPath {
	return PlatformToolsPath(p.path)
}

func (p *PlatformTools) download() error {
	p.logger.Debugf("making directory %v", p.workingDirectory)
	_ = os.Mkdir(p.workingDirectory, os.ModePerm)

	p.logger.Debugf("creating file %v", p.zipFile)
	out, err := os.Create(p.zipFile)
	if err != nil {
		return err
	}
	defer out.Close()

	p.logger.Debugf("downloading %v", p.downloadURI)
	resp, err := p.httpClient.Get(p.downloadURI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad download status from %v: %v", p.zipFile, resp.Status)
	}

	p.logger.Debugf("copying data to %v", p.zipFile)
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (p *PlatformTools) extract() error {
	p.logger.Debugf("unzipping %v to directory %v", p.zipFile, p.workingDirectory)
	return archiver.Unarchive(p.zipFile, p.workingDirectory)
}

func (p *PlatformTools) verify() error {
	p.logger.Debugf("verifying sha256 for %v = %v", p.zipFile, p.sha256)
	f, err := os.Open(p.zipFile)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if p.sha256 != sum {
		return fmt.Errorf("expected sha256 to be %v but got %v", p.sha256, sum)
	}
	return nil
}
