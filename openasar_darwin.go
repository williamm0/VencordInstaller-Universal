//go:build darwin

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	path "path/filepath"
	"strconv"
)

const OpenAsarDownloadLink = "https://github.com/GooseMod/OpenAsar/releases/download/nightly/app.asar"

func FindAsarFile(dir string) (*os.File, error) {
	for _, file := range []string{"_app.asar", "app.asar"} {
		f, err := os.Open(path.Join(dir, file))
		if err != nil {
			continue
		}
		stats, err := f.Stat()
		if err == nil && !stats.IsDir() {
			return f, nil
		}
		_ = f.Close()
	}
	return nil, errors.New("Install at " + dir + " has no asar file")
}

func (di *DiscordInstall) IsOpenAsar() (retBool bool) {
	if di.isOpenAsar != nil {
		return *di.isOpenAsar
	}

	defer func() {
		Log.Debug("Checking if", di.path, "is using OpenAsar:", retBool)
		di.isOpenAsar = &retBool
	}()

	asarFile, err := FindAsarFile(path.Join(di.appPath, ".."))
	if err != nil {
		Log.Error(err.Error())
		return false
	}

	b, err := io.ReadAll(asarFile)
	_ = asarFile.Close()
	if err != nil {
		Log.Error(err.Error())
		return false
	}

	if bytes.Contains(b, []byte("OpenAsar")) {
		return true
	}

	return false
}

// InstallOpenAsar downloads OpenAsar to a temp path, then installs it using a
// single admin elevation. This avoids App Management and FDA TCC restrictions.
func (di *DiscordInstall) InstallOpenAsar() error {
	dir := path.Join(di.appPath, "..")
	asarFile, err := FindAsarFile(dir)
	if err != nil {
		return err
	}
	_ = asarFile.Close()

	res, err := http.Get(OpenAsarDownloadLink)
	if err != nil {
		return err
	} else if res.StatusCode >= 300 {
		return errors.New("Failed to fetch OpenAsar - " + strconv.Itoa(res.StatusCode) + ": " + res.Status)
	}

	tmp, err := os.CreateTemp("", "openasar-*.asar")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err = io.Copy(tmp, res.Body); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	backupPath := path.Join(dir, "app.asar.backup")
	shellCmd := fmt.Sprintf(
		"mv %s %s && cp %s %s && chown $(stat -f '%%Su:%%Sg' %s) %s && rm -f %s",
		shellQuote(asarFile.Name()), shellQuote(backupPath),
		shellQuote(tmpPath), shellQuote(asarFile.Name()),
		shellQuote(backupPath), shellQuote(asarFile.Name()),
		shellQuote(tmpPath),
	)

	Log.Debug("Elevating for OpenAsar install")
	if err := elevate(shellCmd); err != nil {
		return errors.New("OpenAsar install failed. Admin prompt may have been cancelled: " + err.Error())
	}

	di.isOpenAsar = Ptr(true)
	return nil
}

// UninstallOpenAsar restores the original asar from the backup using a single
// admin elevation.
func (di *DiscordInstall) UninstallOpenAsar() error {
	dir := path.Join(di.appPath, "..")

	for _, backupFile := range []string{path.Join(dir, "app.asar.backup"), path.Join(dir, "app.asar.original")} {
		if !ExistsFile(backupFile) {
			continue
		}

		asarFile, err := FindAsarFile(dir)
		if err != nil {
			return err
		}
		_ = asarFile.Close()

		shellCmd := fmt.Sprintf("mv %s %s", shellQuote(backupFile), shellQuote(asarFile.Name()))

		Log.Debug("Elevating for OpenAsar uninstall")
		if err := elevate(shellCmd); err != nil {
			return errors.New("OpenAsar uninstall failed. Admin prompt may have been cancelled: " + err.Error())
		}

		di.isOpenAsar = Ptr(false)
		return nil
	}

	return errors.New("No app.asar.backup. Reinstall Discord")
}
