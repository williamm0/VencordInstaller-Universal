//go:build darwin

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"fmt"
	"os"
	path "path/filepath"
)

// patchAppAsar writes a new app.asar to a temp path, then performs all
// renames inside a single osascript elevation (one password prompt total).
// This bypasses both App Management and Full Disk Access TCC restrictions.
func patchAppAsar(dir string, isSystemElectron bool) error {
	appAsar := path.Join(dir, "app.asar")
	_appAsar := path.Join(dir, "_app.asar")

	tmpAsar, err := os.CreateTemp("", "vencord-*.asar")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpAsar.Name()
	tmpAsar.Close()
	defer os.Remove(tmpPath)

	Log.Debug("Writing custom app.asar to temp path", tmpPath)
	if err := WriteAppAsar(tmpPath, VencordDirectory); err != nil {
		return err
	}

	// Build a single shell command that does all file ops as root in one prompt.
	// Using && so a failure stops the chain; the undo branch restores the backup
	// if cp fails after the initial mv.
	var shellCmd string
	if isSystemElectron {
		from := appAsar + ".unpacked"
		to := _appAsar + ".unpacked"
		shellCmd = fmt.Sprintf(
			"mv %s %s && mv %s %s && { cp %s %s && chown $(stat -f '%%Su:%%Sg' %s) %s || { mv %s %s; mv %s %s; exit 1; }; } && rm -f %s",
			shellQuote(appAsar), shellQuote(_appAsar),
			shellQuote(from), shellQuote(to),
			shellQuote(tmpPath), shellQuote(appAsar),
			shellQuote(_appAsar), shellQuote(appAsar),
			shellQuote(_appAsar), shellQuote(appAsar),
			shellQuote(to), shellQuote(from),
			shellQuote(tmpPath),
		)
	} else {
		shellCmd = fmt.Sprintf(
			"mv %s %s && { cp %s %s && chown $(stat -f '%%Su:%%Sg' %s) %s || { mv %s %s; exit 1; }; } && rm -f %s",
			shellQuote(appAsar), shellQuote(_appAsar),
			shellQuote(tmpPath), shellQuote(appAsar),
			shellQuote(_appAsar), shellQuote(appAsar),
			shellQuote(_appAsar), shellQuote(appAsar),
			shellQuote(tmpPath),
		)
	}

	Log.Debug("Elevating for patch file operations")
	if err := elevate(shellCmd); err != nil {
		return errors.New("Patch failed. Admin prompt may have been cancelled: " + err.Error())
	}
	return nil
}

// unpatchAppAsar restores the original app.asar from _app.asar using a single
// elevated shell command so only one password prompt is shown.
func unpatchAppAsar(dir string, isSystemElectron bool) error {
	appAsar := path.Join(dir, "app.asar")
	appAsarTmp := path.Join(dir, "app.asar.tmp")
	_appAsar := path.Join(dir, "_app.asar")

	var shellCmd string
	if isSystemElectron {
		shellCmd = fmt.Sprintf(
			"mv %s %s && mv %s %s && mv %s %s && rm -f %s",
			shellQuote(appAsar), shellQuote(appAsarTmp),
			shellQuote(_appAsar+".unpacked"), shellQuote(appAsar+".unpacked"),
			shellQuote(_appAsar), shellQuote(appAsar),
			shellQuote(appAsarTmp),
		)
	} else {
		shellCmd = fmt.Sprintf(
			"mv %s %s && mv %s %s && rm -f %s",
			shellQuote(appAsar), shellQuote(appAsarTmp),
			shellQuote(_appAsar), shellQuote(appAsar),
			shellQuote(appAsarTmp),
		)
	}

	Log.Debug("Elevating for unpatch file operations")
	if err := elevate(shellCmd); err != nil {
		return errors.New("Unpatch failed. Admin prompt may have been cancelled: " + err.Error())
	}
	return nil
}
