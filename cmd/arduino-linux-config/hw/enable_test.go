// This file is part of arduino-linux-config.
//
// SPDX-FileCopyrightText: Arduino s.r.l. and/or its affiliated companies
// SPDX-License-Identifier: GPL-3.0-or-later

package hw

import (
	"reflect"
	"slices"
	"testing"

	"github.com/arduino/arduino-linux-config/internal/overlay"
	"github.com/arduino/arduino-linux-config/internal/registry"
	"github.com/arduino/arduino-linux-config/internal/status"
	"github.com/arduino/arduino-linux-config/internal/testutil"
)

func Test_parseArguments(t *testing.T) {
	tests := []struct {
		name        string
		carrierName string
		args        []string
		want        []status.StatusDevice
		wantErr     bool
	}{
		{
			name: "One single configuration",
			args: []string{"display=8-dsi-touch-a"},
			want: []status.StatusDevice{
				{Device: "display", Option: "8-dsi-touch-a"},
			},
			wantErr: false,
		},
		{
			name: "Two configuration in one string",
			args: []string{"display=8-dsi-touch-a,camera0=type1-2lanes"},
			want: []status.StatusDevice{
				{Device: "display", Option: "8-dsi-touch-a"},
				{Device: "camera0", Option: "type1-2lanes"},
			},
			wantErr: false,
		},
		{
			name: "Happy path: multiple arguments and spaces",
			args: []string{" display=8-dsi-touch-a ", " camera0 = type1-2lanes "},
			want: []status.StatusDevice{
				{Device: "display", Option: "8-dsi-touch-a"},
				{Device: "camera0", Option: "type1-2lanes"},
			},
			wantErr: false,
		},
		{
			name: "One option defined with a comma",
			args: []string{"camera0=type1-2lanes,"},
			want: []status.StatusDevice{
				{Device: "camera0", Option: "type1-2lanes"},
			},
			wantErr: false,
		},
		{
			name:    "Error: Values containing more than one equal sign, invalid format",
			args:    []string{"camera0=camera1=type1-2lanes"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Error: missing equals sign, invalid format",
			args:    []string{"camera0"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Error: duplicated device in arguments, invalid format",
			args:    []string{"camera0=type1-2lanes", "camera0=type1-4lanes"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUserArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseArguments() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseArguments() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateUserConfiguration(t *testing.T) {
	builtin := registry.Mount{
		Name: "builtin",
		Kind: registry.KindCarrier,
		Devices: []registry.Device{
			{
				Name:       "display",
				DeviceType: registry.DeviceTypeDisplay,
				Options: []registry.DeviceOption{
					{Name: "none"},
					{Name: "5-dsi-touch-a"},
				},
			},
		},
	}
	withBase := registry.Mount{
		Name:         "media-carrier",
		Kind:         registry.KindCarrier,
		EnabledDtbos: []string{"base.dtbo"},
		Devices:      builtin.Devices,
	}

	tests := []struct {
		name      string
		mount     registry.Mount
		selection []status.StatusDevice
		wantErr   bool
	}{
		{
			name:      "mount without base overlays requires a device option",
			mount:     builtin,
			selection: nil,
			wantErr:   true,
		},
		{
			name:      "mount without base overlays accepts a device option",
			mount:     builtin,
			selection: []status.StatusDevice{{Device: "display", Option: "5-dsi-touch-a"}},
			wantErr:   false,
		},
		{
			name:      "mount with base overlays allows an empty selection",
			mount:     withBase,
			selection: nil,
			wantErr:   false,
		},
		{
			name:      "unknown device is rejected",
			mount:     builtin,
			selection: []status.StatusDevice{{Device: "camera0", Option: "none"}},
			wantErr:   true,
		},
		{
			name:      "unsupported option is rejected",
			mount:     builtin,
			selection: []status.StatusDevice{{Device: "display", Option: "10-dsi-touch-a"}},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUserConfiguration(tt.mount, tt.selection)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateUserConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExampleDeviceOption(t *testing.T) {
	mount := registry.Mount{
		Devices: []registry.Device{
			{
				Name: "display",
				Options: []registry.DeviceOption{
					{Name: "none"},
					{Name: "5-dsi-touch-a"},
				},
			},
		},
	}
	if got := exampleDeviceOption(mount); got != "display=5-dsi-touch-a" {
		t.Errorf("exampleDeviceOption() = %q, want %q", got, "display=5-dsi-touch-a")
	}
}

func TestCollectDtboFiles(t *testing.T) {
	t.Cleanup(testutil.SetupUnoQDebian())

	reg := registry.New()
	mount, exists := reg.FindByName(string(registry.MediaCarrier))
	if !exists {
		t.Fatalf("Failed to initialize production test: MediaCarrier registry not found")
	}

	tests := []struct {
		name          string
		userSelection []status.StatusDevice
		want          []string
	}{
		{
			name: "Camera0 selection without compatibility issues",
			userSelection: []status.StatusDevice{
				{Device: "camera0", Option: "type1-2lanes"},
			},
			want: []string{
				"qrb2210-arduino-imola-carrier-media-camera-imx219-csi0-2lanes.dtbo",
				"qrb2210-arduino-imola-carrier-media.dtbo",
				"qrb2210-arduino-imola-video_sound-usbc.dtbo",
			},
		},
		{
			name: "Incompatible Selection - All devices",
			userSelection: []status.StatusDevice{
				{Device: "camera0", Option: "type1-4lanes"},
				{Device: "camera1", Option: "type1-4lanes"},
				{Device: "display", Option: "8-dsi-touch-a"},
			},
			want: []string{
				"qrb2210-arduino-imola-carrier-media-camera-imx219-csi0-4lanes.dtbo",
				"qrb2210-arduino-imola-carrier-media-camera-imx219-csi1-4lanes.dtbo",
				"qrb2210-arduino-imola-carrier-media-panel-8in_touch_a-dsi.dtbo",
				"qrb2210-arduino-imola-carrier-media.dtbo",
			},
		},
		{
			name: "Incompatible Selection - Touchscreen triggers deletion of video_sound-usbc",
			userSelection: []status.StatusDevice{
				{Device: "display", Option: "8-dsi-touch-a"},
			},
			want: []string{
				"qrb2210-arduino-imola-carrier-media-panel-8in_touch_a-dsi.dtbo",
				"qrb2210-arduino-imola-carrier-media.dtbo",
			},
		},
		{
			name: "Invalid device names are ignored completely",
			userSelection: []status.StatusDevice{
				{Device: "unknown-hw", Option: "none"},
				{Device: "camera0", Option: "invalid-option-str"},
			},
			want: []string{
				"qrb2210-arduino-imola-carrier-media.dtbo",
				"qrb2210-arduino-imola-video_sound-usbc.dtbo",
			},
		},
		{
			name: "Compound Selection - Multiple Devices",
			userSelection: []status.StatusDevice{
				{Device: "camera0", Option: "none"},
				{Device: "camera1", Option: "none"},
			},
			want: []string{
				"qrb2210-arduino-imola-carrier-media.dtbo",
				"qrb2210-arduino-imola-video_sound-usbc.dtbo",
			},
		},
		{
			name: "Touchscreen 10 triggers deletion of video_sound-usbc",
			userSelection: []status.StatusDevice{
				{Device: "display", Option: "10-dsi-touch-a"},
			},
			want: []string{
				"qrb2210-arduino-imola-carrier-media-panel-10in_touch_a-dsi.dtbo",
				"qrb2210-arduino-imola-carrier-media.dtbo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlays, _ := overlay.Collect(mount, tt.userSelection)

			slices.Sort(overlays)
			overlays = slices.Compact(overlays)
			if !slices.Equal(overlays, tt.want) {
				t.Errorf("\nGot:  %v\nWant: %v", overlays, tt.want)
			}
		})
	}
}
