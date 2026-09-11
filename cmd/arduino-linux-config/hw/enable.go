// This file is part of arduino-linux-config.
//
// SPDX-FileCopyrightText: Arduino s.r.l. and/or its affiliated companies
// SPDX-License-Identifier: GPL-3.0-or-later

package hw

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arduino/arduino-linux-config/cmd/arduino-linux-config/dryrun"
	"github.com/arduino/arduino-linux-config/cmd/arduino-linux-config/hw/completion"
	"github.com/arduino/arduino-linux-config/cmd/feedback"
	"github.com/arduino/arduino-linux-config/internal/config"
	"github.com/arduino/arduino-linux-config/internal/devicetree"
	"github.com/arduino/arduino-linux-config/internal/executor"
	"github.com/arduino/arduino-linux-config/internal/registry"
	"github.com/arduino/arduino-linux-config/internal/status"
)

func newEnableCmd(reg registry.Registry, cfg config.Configuration) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "enable <name> [device=option...]",
		Short: "Enable a carrier or a hat, with its device options",
		Example: `  # Configure a media-carrier with an 8 inch display:
  arduino-linux-config hw enable media-carrier display=8-dsi-touch-a

  # Connect the automation hat:
  arduino-linux-config hw enable automation`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if os.Geteuid() != 0 && !dryRun {
				feedback.Fatal("Command 'enable' must be run as root", feedback.ErrPermissionDenied)
			}
			enableHandler(cmd.Context(), reg, cfg, args[0], args[1:], dryRun)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completion.CompleteMountName(reg, toComplete)
			}
			mount, exist := reg.FindByName(args[0])
			if !exist {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return completion.CompleteDeviceOption(mount, args[1:], toComplete)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate the command without applying overlays or writing state")
	return cmd
}

// Since a board reboot can occur asynchronously with the configuration, we must
// track both the current and next states.
func enableHandler(ctx context.Context, reg registry.Registry, cfg config.Configuration, name string, deviceArgs []string, dryRun bool) {
	mount := findMount(reg, name)

	selection, err := parseUserArgs(deviceArgs)
	if err != nil {
		feedback.Fatal(err.Error(), feedback.ErrBadArgument)
	}
	if err := validateUserConfiguration(mount, selection); err != nil {
		feedback.Fatal(err.Error(), feedback.ErrBadArgument)
	}

	// The tool keeps one mount of a kind enabled, so the others are disabled.
	desired := devicetree.Desired{mount.Name: {Enable: true, StatusDevices: selection}}
	for _, other := range reg.ByKind(mount.Kind).Mounts {
		if other.Name != mount.Name {
			desired[other.Name] = status.MountStatus{Enable: false}
		}
	}

	exec, recorder := executor.Real(), executor.NewRecorder()
	if dryRun {
		exec = recorder
	}

	incompatible, err := devicetree.Rebuild(ctx, exec, reg, cfg, desired)
	if err != nil {
		feedback.Fatal(err.Error(), feedback.ErrGeneric)
	}
	if len(incompatible) > 0 {
		feedback.Warnf("Incompatible overlays, removing %v", incompatible)
	}

	if dryRun {
		subject := fmt.Sprintf("%s '%s'", string(mount.Kind), mount.Name)
		feedback.PrintResult(dryrun.Result{Subject: subject, Effects: recorder.Effects()})
		return
	}

	feedback.Warnf("Configuration enabled (will take effect on next boot)")
	// Every mount is shown, because enabling one disables the others of its kind.
	showHandler(cfg, reg, "")
}

func parseUserArgs(args []string) ([]status.StatusDevice, error) {
	selection := make([]status.StatusDevice, 0, len(args))
	for _, arg := range args {
		// Handle "key=val,key2=val2"
		pairs := strings.Split(arg, ",")

		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}

			parts := strings.Split(pair, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid argument %q: expected device=option format", pair)
			}

			deviceName, optionName := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

			if slices.ContainsFunc(selection, func(s status.StatusDevice) bool {
				return s.Device == deviceName
			}) {
				return nil, fmt.Errorf("duplicate device %q in arguments", deviceName)
			}

			selection = append(selection, status.StatusDevice{
				Device: deviceName,
				Option: optionName,
			})

		}
	}

	return selection, nil
}

func validateUserConfiguration(mount registry.Mount, selection []status.StatusDevice) error {
	// A mount without base overlays applies nothing on its own, so enabling it
	// only affects the device tree when the user selects a device option.
	// Refuse a selection-less enable to avoid a no-op command.
	if len(mount.EnabledDtbos) == 0 && len(mount.Devices) > 0 && len(selection) == 0 {
		if example := exampleDeviceOption(mount); example != "" {
			return fmt.Errorf("%s requires a device option, e.g. %q", mount.Name, example)
		}
		return fmt.Errorf("%s requires a device option", mount.Name)
	}

	for _, s := range selection {
		device, exist := mount.FindDeviceByName(registry.DeviceName(s.Device))
		if !exist {
			return fmt.Errorf("unknown device for %s: %q", mount.Name, s.Device)
		}
		if !slices.ContainsFunc(device.Options, func(o registry.DeviceOption) bool { return o.Name == s.Option }) {
			return fmt.Errorf("device %q does not support option %q", s.Device, s.Option)
		}
	}
	return nil
}

// exampleDeviceOption returns a sample "device=option" hint, preferring a real
// option over "none" so the message points at a meaningful selection.
func exampleDeviceOption(mount registry.Mount) string {
	var fallback string
	for _, device := range mount.Devices {
		for _, option := range device.Options {
			if option.Name == string(registry.None) {
				continue
			}
			return fmt.Sprintf("%s=%s", device.Name, option.Name)
		}
		if fallback == "" && len(device.Options) > 0 {
			fallback = fmt.Sprintf("%s=%s", device.Name, device.Options[0].Name)
		}
	}
	return fallback
}
