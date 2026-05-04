/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"github.com/ProtonMail/go-appdir"
	"os"
	"os/exec"
	path "path/filepath"
	"strings"
)


var BaseDir string
var VencordDirectory string

func init() {
	if dir := os.Getenv("VENCORD_USER_DATA_DIR"); dir != "" {
		Log.Debug("Using VENCORD_USER_DATA_DIR")
		BaseDir = dir
	} else if dir = os.Getenv("DISCORD_USER_DATA_DIR"); dir != "" {
		Log.Debug("Using DISCORD_USER_DATA_DIR/../VencordData")
		BaseDir = path.Join(dir, "..", "VencordData")
	} else {
		Log.Debug("Using UserConfig")
		BaseDir = appdir.New("Vencord").UserConfig()
	}

	if dir := os.Getenv("VENCORD_DIRECTORY"); dir != "" {
		Log.Debug("Using VENCORD_DIRECTORY")
		VencordDirectory = dir
	} else {
		VencordDirectory = path.Join(BaseDir, "vencord.asar")
	}
}

type DiscordInstall struct {
	path             string // the base path
	branch           string // canary / stable / ...
	appPath          string // List of app folder to patch
	isPatched        bool
	isFlatpak        bool
	isSystemElectron bool // Needs special care https://aur.archlinux.org/packages/discord_arch_electron
	isOpenAsar       *bool
}

//region Patch

// patchAppAsar and unpatchAppAsar are defined in patcher_ops.go (non-darwin)
// and patcher_darwin.go (darwin) via build tags.

func (di *DiscordInstall) patch() error {
	Log.Info("Patching " + di.path + "...")
	if LatestHash != InstalledHash {
		if err := InstallLatestBuilds(); err != nil {
			return nil // already shown dialog so don't return same error again
		}
	}

	PreparePatch(di)

	if di.isPatched {
		Log.Info(di.path, "is already patched. Unpatching first...")
		if err := di.unpatch(); err != nil {
			if errors.Is(err, os.ErrPermission) {
				return err
			}
			return errors.New("patch: Failed to unpatch already patched install '" + di.path + "':\n" + err.Error())
		}
	}

	if di.isSystemElectron {
		if err := patchAppAsar(di.path, true); err != nil {
			return err
		}
	} else {
		if err := patchAppAsar(path.Join(di.appPath, ".."), false); err != nil {
			return err
		}
	}

	Log.Info("Successfully patched", di.path)
	di.isPatched = true

	if di.isFlatpak {
		pathElements := strings.Split(di.path, "/")
		var name string
		for _, e := range pathElements {
			if strings.HasPrefix(e, "com.discordapp") {
				name = e
				break
			}
		}

		Log.Debug("This is a flatpak. Trying to grant the Flatpak access to", VencordDirectory+"...")

		isSystemFlatpak := strings.HasPrefix(di.path, "/var")
		var args []string
		if !isSystemFlatpak {
			args = append(args, "--user")
		}
		args = append(args, "override", name, "--filesystem="+VencordDirectory)
		fullCmd := "flatpak " + strings.Join(args, " ")

		Log.Debug("Running", fullCmd)

		var err error
		if !isSystemFlatpak && os.Getuid() == 0 {
			// We are operating on a user flatpak but are root
			actualUser := os.Getenv("SUDO_USER")
			Log.Debug("This is a user install but we are root. Using su to run as", actualUser)
			cmd := exec.Command("su", "-", actualUser, "-c", "sh", "-c", fullCmd)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		} else {
			cmd := exec.Command("flatpak", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		}
		if err != nil {
			return errors.New("Failed to grant Discord Flatpak access to " + VencordDirectory + ": " + err.Error())
		}
	}
	return nil
}

//endregion

// region Unpatch

// unpatchAppAsar defined in patcher_ops.go (non-darwin) and patcher_darwin.go (darwin).

func (di *DiscordInstall) unpatch() error {
	Log.Info("Unpatching " + di.path + "...")

	PreparePatch(di)

	if di.isSystemElectron {
		if err := unpatchAppAsar(di.path, true); err != nil {
			return err
		}
	} else {
		if err := unpatchAppAsar(path.Join(di.appPath, ".."), false); err != nil {
			return err
		}
	}

	Log.Info("Successfully unpatched", di.path)
	di.isPatched = false
	return nil
}

//endregion
