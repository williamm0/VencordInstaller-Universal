/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"fmt"
	"os"
	"os/exec"
	path "path/filepath"
	"strings"
)

var macosNames = map[string]string{
	"stable": "Discord.app",
	"ptb":    "Discord PTB.app",
	"canary": "Discord Canary.app",
	"dev":    "Discord Development.app",
}

func ParseDiscord(p, branch string) *DiscordInstall {
	if !ExistsFile(p) {
		return nil
	}

	resources := path.Join(p, "/Contents/Resources")
	if !ExistsFile(resources) {
		return nil
	}

	if branch == "" {
		branch = GetBranch(strings.TrimSuffix(p, ".app"))
	}

	app := path.Join(resources, "app")
	return &DiscordInstall{
		path:             p,
		branch:           branch,
		appPath:          app,
		isPatched:        ExistsFile(path.Join(resources, "_app.asar")),
		isFlatpak:        false,
		isSystemElectron: false,
	}
}

func FindDiscords() []any {
	var discords []any
	bases := []string{
		"/Applications",
		path.Join(os.Getenv("HOME"), "Applications"),
	}
	for branch, dirname := range macosNames {
		for _, base := range bases {
			p := path.Join(base, dirname)
			if discord := ParseDiscord(p, branch); discord != nil {
				Log.Debug("Found Discord Install at", p)
				discords = append(discords, discord)
			}
		}
	}
	return discords
}

// elevate runs a shell command via osascript with administrator privileges.
// This shows the native macOS password dialog and avoids relying on FDA.
func elevate(shellCmd string) error {
	script := fmt.Sprintf(`do shell script "%s" with administrator privileges`, strings.ReplaceAll(shellCmd, `"`, `\"`))
	return exec.Command("osascript", "-e", script).Run()
}

// shellQuote wraps a path in single quotes, escaping any embedded single quotes.
func shellQuote(p string) string {
	return "'" + strings.ReplaceAll(p, "'", "'\\''") + "'"
}

// PreparePatch fixes ownership and write permissions on the Discord Resources
// directory using an admin elevation dialog so FDA is not needed.
func PreparePatch(di *DiscordInstall) {
	resourcesDir := path.Join(di.appPath, "..")
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "$(id -un)"
	}
	quoted := shellQuote(resourcesDir)
	shellCmd := fmt.Sprintf("chown -R %s:staff %s && chmod -R u+w %s", currentUser, quoted, quoted)
	if err := elevate(shellCmd); err != nil {
		Log.Warn("PreparePatch elevation prompt failed or was cancelled:", err)
	}
}

func FixOwnership(p string) error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "$(id -un)"
	}
	quoted := shellQuote(p)
	shellCmd := fmt.Sprintf("chown -R %s:staff %s && chmod -R u+w %s", currentUser, quoted, quoted)
	return elevate(shellCmd)
}

func CheckScuffedInstall() bool {
	return false
}
