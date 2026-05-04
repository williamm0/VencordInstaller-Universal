# VencordInstaller Universal

A fork of [Vencord/Installer](https://github.com/Vencord/Installer) that runs natively on both Apple Silicon and Intel Macs, with a full fix for the App Management permission errors that block the official build on macOS 13 (Ventura) and later.
<img width="2500" height="1080" alt="Vencord-Universal" src="https://github.com/user-attachments/assets/4de4b4d7-aa31-4f4b-b2d9-bb3148e6fe69" />

## What is this

[Vencord](https://github.com/Vendicated/Vencord) is a Discord client mod. This installer puts it in and takes it out. The official installer ships Intel-only and breaks on modern macOS due to App Management restrictions. This fork fixes both.

## Download

**[Releases](https://github.com/williamm0/VencordInstaller-Universal/releases/latest)**

Download `VencordInstaller.MacOS.universal.zip`, unzip it, and move `VencordInstaller.app` to `/Applications`.

> If macOS says the app is damaged or from an unidentified developer, right-click the app and choose Open, then click Open again in the dialog.

## What is fixed

**Universal binary.** The official release is Intel-only and runs under Rosetta 2 on Apple Silicon. This build is a fat binary containing both `arm64` and `x86_64` slices so it runs natively on both.

**App Management permission errors on macOS 13+.** Apple introduced the App Management privacy category in macOS Ventura. It blocks any process that does not have an explicit TCC grant from modifying apps in `/Applications`, even as root. The official installer asks for Full Disk Access, but FDA is not sufficient and unsigned apps cannot reliably receive either grant.

This fork routes all install and uninstall operations through `Terminal.app`, which has App Management and Full Disk Access permissions by default. When you click Install or Uninstall:

1. A Terminal window opens
2. You may be prompted for your Mac password
3. The operations run, the window closes automatically
4. The installer shows the result

No changes to System Settings are required. If you previously added VencordInstaller to Full Disk Access or App Management, you can remove it.

## Building from source

Requires [Go](https://go.dev/doc/install) and `pkg-config` + `sdl2` (via Homebrew on macOS).

```sh
brew install pkg-config sdl2
go mod tidy
```

### macOS universal binary

```sh
# arm64
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
  go build -tags static -o VencordInstaller_arm64

# amd64
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
  CGO_CFLAGS="-arch x86_64" CGO_LDFLAGS="-arch x86_64" \
  go build -tags static -o VencordInstaller_amd64

# combine
lipo -create VencordInstaller_arm64 VencordInstaller_amd64 \
  -output VencordInstaller
```

Then package it:

```sh
mkdir -p VencordInstaller.app/Contents/{MacOS,Resources}
cp macos/Info.plist VencordInstaller.app/Contents/
cp VencordInstaller    VencordInstaller.app/Contents/MacOS/
cp macos/icon.icns     VencordInstaller.app/Contents/Resources/
zip -r VencordInstaller.MacOS.universal.zip VencordInstaller.app
```

### Other platforms

See the [GitHub Actions workflow](.github/workflows/release.yml) for Linux and Windows build steps. Linux and Windows builds are unchanged from upstream.

## Credits and copyright

**VencordInstaller** is created and maintained by [Vendicated](https://github.com/Vendicated) and the [Vencord contributors](https://github.com/Vencord/Installer/graphs/contributors).

**Vencord** is created and maintained by [Vendicated](https://github.com/Vendicated) and the [Vencord contributors](https://github.com/Vendicated/Vencord/graphs/contributors).

All original source code is copyright (c) 2023 Vendicated and Vencord contributors and is licensed under the [GNU General Public License v3.0](LICENSE).

This fork is maintained by [williamm0](https://github.com/williamm0) and contributes only the macOS-specific changes described above. No claim is made over the original work. All credit for the installer and Vencord itself belongs to the original authors.
