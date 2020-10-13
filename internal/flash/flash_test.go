package flash

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.com/calyxos/device-flasher/internal/factoryimage"
	"gitlab.com/calyxos/device-flasher/internal/flash/mocks"
	"gitlab.com/calyxos/device-flasher/internal/platformtools"
	"gitlab.com/calyxos/device-flasher/internal/platformtools/fastboot"
	"testing"
)

var (
	testOS = "TestOS"
	testDevice = &Device{ID: "8AAY0GK9A", Codename: "crosshatch"}
)

func TestFlash(t *testing.T) {
	ctrl := gomock.NewController(t)

	tests := map[string]struct {
		device *Device
		prepare func(*mocks.MockFactoryImageFlasher, *mocks.MockPlatformToolsFlasher,
			*mocks.MockADBFlasher, *mocks.MockFastbootFlasher, *mocks.MockUdevFlasher)
		expectedErr error
	}{
		"happy path flash successful": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(nil)
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Locked, nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Unlocked).Return(nil)
				mockPlatformTools.EXPECT().Path().Return(platformtools.PlatformToolsPath("/tmp"))
				mockFactoryImage.EXPECT().FlashAll(platformtools.PlatformToolsPath("/tmp")).Return(nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Locked).Return(nil)
				mockFastboot.EXPECT().Reboot(testDevice.ID).Return(nil)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: nil,
		},
		"unlocked device skips unlocking step": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(nil)
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Unlocked, nil)
				mockPlatformTools.EXPECT().Path().Return(platformtools.PlatformToolsPath("/tmp"))
				mockFactoryImage.EXPECT().FlashAll(platformtools.PlatformToolsPath("/tmp")).Return(nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Locked).Return(nil)
				mockFastboot.EXPECT().Reboot(testDevice.ID).Return(nil)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: nil,
		},
		"factory image validation failure": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(factoryimage.ErrorValidation)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: factoryimage.ErrorValidation,
		},
		"adb reboot bootloader error not fatal": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(errors.New("not fatal"))
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Locked, nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Unlocked).Return(nil)
				mockPlatformTools.EXPECT().Path().Return(platformtools.PlatformToolsPath("/tmp"))
				mockFactoryImage.EXPECT().FlashAll(platformtools.PlatformToolsPath("/tmp")).Return(nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Locked).Return(nil)
				mockFastboot.EXPECT().Reboot(testDevice.ID).Return(nil)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: nil,
		},
		"get bootloader status failure": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(nil)
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Unknown, fastboot.ErrorCommandFailure)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: fastboot.ErrorCommandFailure,
		},
		"set bootloader status unlock failure": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(nil)
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Locked, nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Unlocked).Return(fastboot.ErrorUnlockBootloader)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: fastboot.ErrorUnlockBootloader,
		},
		"flash all error": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(nil)
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Locked, nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Unlocked).Return(nil)
				mockPlatformTools.EXPECT().Path().Return(platformtools.PlatformToolsPath("/tmp"))
				mockFactoryImage.EXPECT().FlashAll(platformtools.PlatformToolsPath("/tmp")).Return(factoryimage.ErrorFailedToFlash)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: factoryimage.ErrorFailedToFlash,
		},
		"set bootloader status lock failure": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(nil)
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Locked, nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Unlocked).Return(nil)
				mockPlatformTools.EXPECT().Path().Return(platformtools.PlatformToolsPath("/tmp"))
				mockFactoryImage.EXPECT().FlashAll(platformtools.PlatformToolsPath("/tmp")).Return(nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Locked).Return(fastboot.ErrorLockBootloader)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: fastboot.ErrorLockBootloader,
		},
		"reboot failure": {
			device: testDevice,
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockFactoryImage.EXPECT().Validate(testDevice.Codename).Return(nil)
				mockADB.EXPECT().RebootIntoBootloader(testDevice.ID).Return(nil)
				mockFastboot.EXPECT().GetBootloaderLockStatus(testDevice.ID).Return(fastboot.Locked, nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Unlocked).Return(nil)
				mockPlatformTools.EXPECT().Path().Return(platformtools.PlatformToolsPath("/tmp"))
				mockFactoryImage.EXPECT().FlashAll(platformtools.PlatformToolsPath("/tmp")).Return(nil)
				mockFastboot.EXPECT().SetBootloaderLockStatus(testDevice.ID, fastboot.Locked).Return(nil)
				mockFastboot.EXPECT().Reboot(testDevice.ID).Return(fastboot.ErrorRebootFailure)
				mockADB.EXPECT().KillServer()
			},
			expectedErr: fastboot.ErrorRebootFailure,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockFactoryImage := mocks.NewMockFactoryImageFlasher(ctrl)
			mockPlatformTools := mocks.NewMockPlatformToolsFlasher(ctrl)
			mockADB := mocks.NewMockADBFlasher(ctrl)
			mockFastboot := mocks.NewMockFastbootFlasher(ctrl)
			mockUdev := mocks.NewMockUdevFlasher(ctrl)

			if tc.prepare != nil {
				tc.prepare(mockFactoryImage, mockPlatformTools, mockADB, mockFastboot, mockUdev)
			}

			flash := New(&Config{
				HostOS: testOS,
				FactoryImage: mockFactoryImage,
				PlatformTools: mockPlatformTools,
				ADB: mockADB,
				Fastboot: mockFastboot,
				Udev: mockUdev,
				Logger: logrus.StandardLogger(),
			})

			err := flash.Flash(tc.device)
			if tc.expectedErr == nil {
				assert.Nil(t, err)
			} else {
				assert.True(t, errors.Is(err, tc.expectedErr), true)
			}
		})
	}
}

func TestDiscoverDevices(t *testing.T) {
	ctrl := gomock.NewController(t)

	tests := map[string]struct {
		device *Device
		prepare func(*mocks.MockFactoryImageFlasher, *mocks.MockPlatformToolsFlasher,
			*mocks.MockADBFlasher, *mocks.MockFastbootFlasher, *mocks.MockUdevFlasher)
		expectedErr error
		expectedDevices []*Device
	}{
		"discovery successful with default device": {
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
							mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockUdev.EXPECT().Setup()
				mockADB.EXPECT().GetDeviceIds().Return([]string{testDevice.ID}, nil)
				mockADB.EXPECT().GetDeviceCodename(testDevice.ID).Return(testDevice.Codename, nil)
			},
			expectedErr: nil,
			expectedDevices: []*Device{testDevice},
		},
		"discovery fails when get device fails for both adb and fastboot": {
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockUdev.EXPECT().Setup()
				mockADB.EXPECT().GetDeviceIds().Return(nil, errors.New("failed"))
				mockFastboot.EXPECT().GetDeviceIds().Return(nil, errors.New("failed"))
			},
			expectedErr: ErrNoDevicesFound,
			expectedDevices: nil,
		},
		"discovery successful if adb fails and then fastboot succeeds": {
			prepare: func(mockFactoryImage *mocks.MockFactoryImageFlasher, mockPlatformTools *mocks.MockPlatformToolsFlasher,
				mockADB *mocks.MockADBFlasher, mockFastboot *mocks.MockFastbootFlasher, mockUdev *mocks.MockUdevFlasher) {
				mockUdev.EXPECT().Setup()
				mockADB.EXPECT().GetDeviceIds().Return(nil, errors.New("failed"))
				mockADB.EXPECT().GetDeviceCodename(testDevice.ID).Return("", errors.New("failed"))
				mockFastboot.EXPECT().GetDeviceIds().Return([]string{testDevice.ID}, nil)
				mockFastboot.EXPECT().GetDeviceCodename(testDevice.ID).Return(testDevice.Codename, nil)
			},
			expectedErr: nil,
			expectedDevices: []*Device{testDevice},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockFactoryImage := mocks.NewMockFactoryImageFlasher(ctrl)
			mockPlatformTools := mocks.NewMockPlatformToolsFlasher(ctrl)
			mockADB := mocks.NewMockADBFlasher(ctrl)
			mockFastboot := mocks.NewMockFastbootFlasher(ctrl)
			mockUdev := mocks.NewMockUdevFlasher(ctrl)

			if tc.prepare != nil {
				tc.prepare(mockFactoryImage, mockPlatformTools, mockADB, mockFastboot, mockUdev)
			}

			flash := New(&Config{
				HostOS: testOS,
				FactoryImage: mockFactoryImage,
				PlatformTools: mockPlatformTools,
				ADB: mockADB,
				Fastboot: mockFastboot,
				Udev: mockUdev,
				Logger: logrus.StandardLogger(),
			})

			devices, err := flash.DiscoverDevices()
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedDevices, devices)
		})
	}
}