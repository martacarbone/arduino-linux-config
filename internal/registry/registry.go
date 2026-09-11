// This file is part of arduino-linux-config.
//
// SPDX-FileCopyrightText: Arduino s.r.l. and/or its affiliated companies
// SPDX-License-Identifier: GPL-3.0-or-later

package registry

import (
	"github.com/arduino/go-paths-helper"

	"github.com/arduino/arduino-linux-config/internal/config"
)

type DeviceType string

const (
	DeviceTypeCamera  DeviceType = "camera"
	DeviceTypeDisplay DeviceType = "display"
)

type Registry struct {
	Mounts []Mount
}

// Mount names are unique over every kind, so the name alone selects a part.
func (r Registry) FindByName(name string) (Mount, bool) {
	for _, m := range r.Mounts {
		if string(m.Name) == name {
			return m, true
		}
	}
	return Mount{}, false
}

// ByKind returns every mount when kind is empty.
func (r Registry) ByKind(kind Kind) Registry {
	mounts := make([]Mount, 0, len(r.Mounts))
	for _, m := range r.Mounts {
		if kind == "" || m.Kind == kind {
			mounts = append(mounts, m)
		}
	}
	return Registry{Mounts: mounts}
}

// Kind groups the mounts by the connector they use.
type Kind string

const (
	KindCarrier Kind = "carrier"
	KindHat     Kind = "hat"
)

type DeviceName string

const (
	None    DeviceName = "none"
	Camera0 DeviceName = "camera0"
	Camera1 DeviceName = "camera1"
	Display DeviceName = "display"
)

type MountName string

const (
	MediaCarrier   MountName = "media-carrier"
	AudioCodecZero MountName = "audio-codec-zero"
	Automation     MountName = "automation"
	Builtin        MountName = "builtin"
)

// Mount is a part that plugs into the board and adds device tree overlays.
// A carrier and a hat differ only by Kind and by the connector they use.
type Mount struct {
	Name          MountName
	Kind          Kind
	EnabledDtbos  []string
	DisabledDtbos []string
	Devices       []Device // empty for the hats available today
}

func (c Mount) FindDeviceByName(deviceName DeviceName) (Device, bool) {
	for _, d := range c.Devices {
		if d.Name == deviceName {
			return d, true
		}
	}
	return Device{}, false
}

// Device represents a configurable hardware device on a mount
type Device struct {
	Name       DeviceName
	DeviceType DeviceType
	Options    []DeviceOption
}

// DeviceOption represents a configuration option for a device
type DeviceOption struct {
	Name             string
	DtboFiles        []string
	IncompatibleDtbo []string
}

func New() Registry {
	board := config.GetBoardID()
	boardOs := config.GetLinuxDistribution()

	switch {
	case board == "unoq":
		// unoq has no hat connector, so it declares no mount of kind hat.
		return Registry{
			Mounts: []Mount{unoqMediaCarrier},
		}
	case board == "ventunoq" && boardOs == "ubuntu":
		createFakeFiles()
		mounts := make([]Mount, 0, len(ventunoqUbuntuHats)+2)
		mounts = append(mounts, ventunoqUbuntuHats...)
		mounts = append(mounts, unoqMediaCarrier)
		mounts = append(mounts, ventunoBuiltin)
		return Registry{Mounts: mounts}
	default:
		return Registry{}
	}
}

var unoqMediaCarrier = Mount{
	Name: MediaCarrier,
	Kind: KindCarrier,
	EnabledDtbos: []string{
		"qrb2210-arduino-imola-carrier-media.dtbo",
		"qrb2210-arduino-imola-video_sound-usbc.dtbo",
	},
	DisabledDtbos: []string{
		"qrb2210-arduino-imola-video_sound-usbc.dtbo",
	},
	Devices: []Device{
		{
			Name:       "camera0",
			DeviceType: DeviceTypeCamera,
			Options: []DeviceOption{
				{
					Name:      "none",
					DtboFiles: []string{"qrb2210-arduino-imola-video_sound-usbc.dtbo"},
				},
				{
					Name: "type1-2lanes",
					DtboFiles: []string{
						"qrb2210-arduino-imola-video_sound-usbc.dtbo",
						"qrb2210-arduino-imola-carrier-media.dtbo",
						"qrb2210-arduino-imola-carrier-media-camera-imx219-csi0-2lanes.dtbo",
					},
				},
				{
					Name: "type1-4lanes",
					DtboFiles: []string{
						"qrb2210-arduino-imola-video_sound-usbc.dtbo",
						"qrb2210-arduino-imola-carrier-media.dtbo",
						"qrb2210-arduino-imola-carrier-media-camera-imx219-csi0-4lanes.dtbo",
					},
				},
			},
		},
		{
			Name:       "camera1",
			DeviceType: DeviceTypeCamera,
			Options: []DeviceOption{
				{
					Name:      "none",
					DtboFiles: []string{"qrb2210-arduino-imola-video_sound-usbc.dtbo"},
				},
				{
					Name: "type1-2lanes",
					DtboFiles: []string{
						"qrb2210-arduino-imola-video_sound-usbc.dtbo",
						"qrb2210-arduino-imola-carrier-media.dtbo",
						"qrb2210-arduino-imola-carrier-media-camera-imx219-csi1-2lanes.dtbo",
					},
				},
				{
					Name: "type1-4lanes",
					DtboFiles: []string{
						"qrb2210-arduino-imola-video_sound-usbc.dtbo",
						"qrb2210-arduino-imola-carrier-media.dtbo",
						"qrb2210-arduino-imola-carrier-media-camera-imx219-csi1-4lanes.dtbo",
					},
				},
			},
		},
		{
			Name:       "display",
			DeviceType: DeviceTypeDisplay,
			Options: []DeviceOption{
				{
					Name:      "none",
					DtboFiles: []string{"qrb2210-arduino-imola-video_sound-usbc.dtbo"},
				},
				{
					Name: "5-dsi-touch-a",
					DtboFiles: []string{
						"qrb2210-arduino-imola-carrier-media.dtbo",
						"qrb2210-arduino-imola-carrier-media-panel-5in_touch_a-dsi.dtbo",
					},
					IncompatibleDtbo: []string{
						"qrb2210-arduino-imola-video_sound-usbc.dtbo",
					},
				},
				{
					Name: "8-dsi-touch-a",
					DtboFiles: []string{
						"qrb2210-arduino-imola-carrier-media.dtbo",
						"qrb2210-arduino-imola-carrier-media-panel-8in_touch_a-dsi.dtbo",
					},
					IncompatibleDtbo: []string{
						"qrb2210-arduino-imola-video_sound-usbc.dtbo",
					},
				},
				{
					Name: "10-dsi-touch-a",
					DtboFiles: []string{
						"qrb2210-arduino-imola-carrier-media.dtbo",
						"qrb2210-arduino-imola-carrier-media-panel-10in_touch_a-dsi.dtbo",
					},
					IncompatibleDtbo: []string{
						"qrb2210-arduino-imola-video_sound-usbc.dtbo",
					},
				},
			},
		},
	},
}

var ventunoqUbuntuHats = []Mount{
	{
		Name: AudioCodecZero,
		Kind: KindHat,
		EnabledDtbos: []string{
			"monaco-addons-iqaudio-codeczero-monza.dtbo",
		},
	},
	{
		Name: Automation,
		Kind: KindHat,
		EnabledDtbos: []string{
			"monaco-monza-automation-hat.dtbo",
		},
	},
}

var ventunoBuiltin = Mount{
	Name: Builtin,
	Kind: KindCarrier,
	Devices: []Device{
		{
			Name:       "display",
			DeviceType: DeviceTypeDisplay,
			Options: []DeviceOption{
				{
					Name:      "none",
					DtboFiles: []string{"monaco-ubuntu-hat.dtbo"},
				},
				{
					Name: "5-dsi-touch-a",
					DtboFiles: []string{
						"monaco-ubutu-automation-hat.dtbo",
					},
				},
			},
		},
	},
}

const (
	fakeOverlaysDir   = "/var/lib/arduino-linux-config/overlays"
	fakeOverlaySource = "monaco-monza-automation-hat.dtbo"
)

// createFakeFiles populates the overlays directory with placeholder DTBO files
// by copying an existing overlay onto every DTBO referenced by the media carrier
// (the qrb* overlays) and the builtin mount (the monaco-ub* files). Files that
// already exist are left untouched, so the call is idempotent and never fails
// when the placeholders are already in place.
func createFakeFiles() {
	overlaysDir := paths.New(fakeOverlaysDir)
	source := overlaysDir.Join(fakeOverlaySource)

	seen := map[string]struct{}{}
	for _, m := range []Mount{unoqMediaCarrier, ventunoBuiltin} {
		for _, dtbo := range mountDtboFiles(m) {
			if _, ok := seen[dtbo]; ok {
				continue
			}
			seen[dtbo] = struct{}{}

			dst := overlaysDir.Join(dtbo)
			if dst.Exist() {
				continue
			}
			// Best effort: a failed copy surfaces later as an apply error.
			_ = source.CopyTo(dst)
		}
	}
}

// mountDtboFiles returns every DTBO file referenced by a mount, across its
// enabled/disabled lists and all of its device options.
func mountDtboFiles(m Mount) []string {
	files := make([]string, 0)
	files = append(files, m.EnabledDtbos...)
	files = append(files, m.DisabledDtbos...)
	for _, d := range m.Devices {
		for _, o := range d.Options {
			files = append(files, o.DtboFiles...)
			files = append(files, o.IncompatibleDtbo...)
		}
	}
	return files
}
