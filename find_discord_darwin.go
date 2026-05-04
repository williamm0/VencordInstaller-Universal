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
	"time"
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

const (
	// fixScriptPath holds the shell script written before each operation.
	fixScriptPath = "/tmp/vencord-fix.sh"
	// successMarkerPath is created by the script on success; absence means failure.
	successMarkerPath = "/tmp/vencord-done"
)

// elevate runs shellCmd as root via Terminal.app.
//
// Terminal.app has the App Management and Full Disk Access TCC permissions
// that are required to modify app bundles in /Applications on macOS 13+.
// The osascript "with administrator privileges" approach does NOT work because
// the root shell it spawns is still subject to App Management restrictions.
//
// The script is written to fixScriptPath and a success marker is written to
// successMarkerPath when the script finishes cleanly. We poll for that marker
// from Go so no Terminal output needs to be parsed.
func elevate(shellCmd string) error {
	os.Remove(successMarkerPath)

	script := "#!/bin/sh\nset -e\n" + shellCmd + "\ntouch " + successMarkerPath + "\n"
	if err := os.WriteFile(fixScriptPath, []byte(script), 0700); err != nil {
		return fmt.Errorf("failed to write elevation script: %w", err)
	}

	// Open a Terminal window that runs the script with sudo.
	// When the sudo command exits (success or failure), the shell returns to
	// the prompt and "is busy" becomes false, ending the repeat loop.
	// The window is then closed automatically.
	osa := `tell application "Terminal"
	activate
	set w to do script "sudo sh '/tmp/vencord-fix.sh'; exit"
	repeat while w is busy
		delay 0.5
	end repeat
	delay 0.5
	try
		close (window of w) without saving
	end try
end tell`

	if err := exec.Command("osascript", "-e", osa).Run(); err != nil {
		return fmt.Errorf("could not open Terminal: %w", err)
	}

	// Poll for the marker the script writes on clean completion.
	// Typical completion is well under 10 seconds; 5 minutes is the hard limit.
	for i := 0; i < 600; i++ {
		time.Sleep(500 * time.Millisecond)
		if _, err := os.Stat(successMarkerPath); err == nil {
			os.Remove(successMarkerPath)
			os.Remove(fixScriptPath)
			return nil
		}
	}

	return fmt.Errorf("timed out waiting for operation to finish. Make sure Discord is fully closed, then try again")
}

// shellQuote wraps p in single quotes, escaping any embedded single quotes.
func shellQuote(p string) string {
	return "'" + strings.ReplaceAll(p, "'", "'\\''") + "'"
}

// PreparePatch is a no-op on macOS; elevation is handled per-operation.
func PreparePatch(_ *DiscordInstall) {}

func FixOwnership(_ string) error { return nil }

func CheckScuffedInstall() bool { return false }
