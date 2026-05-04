//go:build darwin

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"fmt"
	"os"
	path "path/filepath"
)

// patchAppAsar writes the Vencord loader asar to a temp path (no permissions
// needed for /tmp), then runs a single elevated shell script via Terminal.app
// that renames the original app.asar to _app.asar and puts the new one in
// place. A trap-based cleanup restores the original if any step fails.
func patchAppAsar(dir string, isSystemElectron bool) error {
	appAsar := path.Join(dir, "app.asar")
	_appAsar := path.Join(dir, "_app.asar")

	tmp, err := os.CreateTemp("", "vencord-*.asar")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	Log.Debug("Writing Vencord asar to", tmpPath)
	if err := WriteAppAsar(tmpPath, VencordDirectory); err != nil {
		return err
	}

	var shellCmd string
	if isSystemElectron {
		unpackedOrig := appAsar + ".unpacked"
		unpackedBak := _appAsar + ".unpacked"
		shellCmd = fmt.Sprintf(`
ORIG=%s
BACKUP=%s
UNPACKED_ORIG=%s
UNPACKED_BACKUP=%s
TEMP=%s
UNDO=1
cleanup() {
    if [ "$UNDO" = "1" ]; then
        [ -f "$BACKUP" ] && [ ! -f "$ORIG" ] && mv "$BACKUP" "$ORIG" 2>/dev/null || true
        [ -f "$UNPACKED_BACKUP" ] && [ ! -f "$UNPACKED_ORIG" ] && mv "$UNPACKED_BACKUP" "$UNPACKED_ORIG" 2>/dev/null || true
    fi
    rm -f "$TEMP"
}
trap cleanup EXIT
mv "$ORIG" "$BACKUP"
mv "$UNPACKED_ORIG" "$UNPACKED_BACKUP"
cp "$TEMP" "$ORIG"
chown "$(stat -f '%%Su:%%Sg' "$BACKUP")" "$ORIG"
UNDO=0`,
			shellQuote(appAsar), shellQuote(_appAsar),
			shellQuote(unpackedOrig), shellQuote(unpackedBak),
			shellQuote(tmpPath),
		)
	} else {
		shellCmd = fmt.Sprintf(`
ORIG=%s
BACKUP=%s
TEMP=%s
UNDO=1
cleanup() {
    if [ "$UNDO" = "1" ]; then
        [ -f "$BACKUP" ] && [ ! -f "$ORIG" ] && mv "$BACKUP" "$ORIG" 2>/dev/null || true
    fi
    rm -f "$TEMP"
}
trap cleanup EXIT
mv "$ORIG" "$BACKUP"
cp "$TEMP" "$ORIG"
chown "$(stat -f '%%Su:%%Sg' "$BACKUP")" "$ORIG"
UNDO=0`,
			shellQuote(appAsar), shellQuote(_appAsar), shellQuote(tmpPath),
		)
	}

	if err := elevate(shellCmd); err != nil {
		return fmt.Errorf("patch failed: %w", err)
	}
	return nil
}

// unpatchAppAsar restores the original app.asar from _app.asar via Terminal.
func unpatchAppAsar(dir string, isSystemElectron bool) error {
	appAsar := path.Join(dir, "app.asar")
	appAsarTmp := path.Join(dir, "app.asar.tmp")
	_appAsar := path.Join(dir, "_app.asar")

	var shellCmd string
	if isSystemElectron {
		shellCmd = fmt.Sprintf(`
ORIG=%s
TMP=%s
BACKUP=%s
UNPACKED_ORIG=%s
UNPACKED_BACKUP=%s
UNDO=1
cleanup() {
    if [ "$UNDO" = "1" ]; then
        [ -f "$TMP" ] && [ ! -f "$ORIG" ] && mv "$TMP" "$ORIG" 2>/dev/null || true
        rm -f "$TMP"
    else
        rm -f "$TMP"
    fi
}
trap cleanup EXIT
mv "$ORIG" "$TMP"
mv "$UNPACKED_BACKUP" "$UNPACKED_ORIG"
mv "$BACKUP" "$ORIG"
UNDO=0`,
			shellQuote(appAsar), shellQuote(appAsarTmp), shellQuote(_appAsar),
			shellQuote(appAsar+".unpacked"), shellQuote(_appAsar+".unpacked"),
		)
	} else {
		shellCmd = fmt.Sprintf(`
ORIG=%s
TMP=%s
BACKUP=%s
UNDO=1
cleanup() {
    if [ "$UNDO" = "1" ]; then
        [ -f "$TMP" ] && [ ! -f "$ORIG" ] && mv "$TMP" "$ORIG" 2>/dev/null || true
        rm -f "$TMP"
    else
        rm -f "$TMP"
    fi
}
trap cleanup EXIT
mv "$ORIG" "$TMP"
mv "$BACKUP" "$ORIG"
UNDO=0`,
			shellQuote(appAsar), shellQuote(appAsarTmp), shellQuote(_appAsar),
		)
	}

	if err := elevate(shellCmd); err != nil {
		return fmt.Errorf("unpatch failed: %w", err)
	}
	return nil
}
