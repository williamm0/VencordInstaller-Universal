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

// fixScriptPath is a known path left on disk when elevation fails so the user
// can run it manually from Terminal: sudo sh /tmp/vencord-fix.sh
const fixScriptPath = "/tmp/vencord-fix.sh"

// elevate writes shellCmd to a temp script file, then runs it via osascript
// with administrator privileges. Writing to a file avoids quoting the whole
// command inside the AppleScript string and lets us capture real stderr.
// On failure the script is kept at fixScriptPath for manual Terminal fallback.
func elevate(shellCmd string) error {
	script := "#!/bin/sh\nset -e\n" + shellCmd + "\n"
	if err := os.WriteFile(fixScriptPath, []byte(script), 0700); err != nil {
		return fmt.Errorf("failed to write elevation script: %w", err)
	}

	osa := `do shell script "sh ` + shellQuote(fixScriptPath) + `" with administrator privileges`
	cmd := exec.Command("osascript", "-e", osa)
	out, runErr := cmd.CombinedOutput()

	if runErr == nil {
		os.Remove(fixScriptPath)
		return nil
	}

	msg := strings.TrimSpace(string(out))

	// -128 is AppleScript's user-cancelled error code
	if strings.Contains(msg, "-128") || strings.Contains(strings.ToLower(msg), "cancel") {
		os.Remove(fixScriptPath)
		return errors.New("password prompt was cancelled")
	}

	// Keep the script for manual fallback and include real error + instructions
	if msg == "" {
		msg = runErr.Error()
	}
	return fmt.Errorf("%s\n\nIf you see 'Operation not permitted', App Management is blocking this.\nRun manually in Terminal:\n  sudo sh %s", msg, fixScriptPath)
}

// shellQuote wraps a path in single quotes, escaping any embedded single quotes.
func shellQuote(p string) string {
	return "'" + strings.ReplaceAll(p, "'", "'\\''") + "'"
}

// PreparePatch is a no-op on macOS. Elevation is handled inside each
// file operation (patchAppAsar, unpatchAppAsar, Install/UninstallOpenAsar)
// via a single osascript admin prompt, bypassing App Management and FDA.
func PreparePatch(_ *DiscordInstall) {}

func FixOwnership(_ string) error {
	return nil
}

func CheckScuffedInstall() bool {
	return false
}
