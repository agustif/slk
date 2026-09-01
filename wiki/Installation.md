# Installation

Install **this fork** ([agustif/slk](https://github.com/agustif/slk)). Commands that mention `gammons/slk`, `gammons/tap`, or AUR `slk` install [upstream](https://github.com/gammons/slk) instead.

**No GitHub Release or semver tag yet** (see [[Gaps]]). Use Homebrew `--HEAD` or Go `@main`. `@latest` is not a fork version until a tag exists.

## Homebrew (macOS and Linux)

The formula lives in a dedicated tap, [agustif/homebrew-tap](https://github.com/agustif/homebrew-tap) (`brew tap agustif/tap` clones that repo). It is **not** [gammons/tap](https://github.com/gammons/homebrew-tap).

```bash
# Drop the upstream cask if it is already installed (same binary name):
brew uninstall --cask slk 2>/dev/null || true

brew install --HEAD agustif/tap/slk
```

That auto-taps `agustif/homebrew-tap` and builds this fork’s `main` from source.

Update later with `brew upgrade --fetch-HEAD slk`.

## Arch Linux

AUR [`slk`](https://aur.archlinux.org/packages/slk) is **upstream**. This fork ships an in-tree `slk-git` PKGBUILD (`packaging/aur`); it is **not published** to the AUR:

```bash
git clone https://github.com/agustif/slk.git
cd slk/packaging/aur
makepkg -si
```

That provides `slk` and conflicts with the upstream AUR package.

## Go

```bash
go install -ldflags="-s -w" -trimpath github.com/agustif/slk/cmd/slk@main
```

The module path is `github.com/agustif/slk`. The binary lands in `$(go env GOPATH)/bin` (usually `~/go/bin`).

## Build from source

Requires Go 1.22+.

On Linux, `Ctrl+V` paste-to-upload needs slightly different setup depending on your session type.

**X11 sessions** use the `golang.design/x/clipboard` library, which requires X11 development headers at build time:

- Debian/Ubuntu: `sudo apt-get install -y libx11-dev`
- Fedora/RHEL: `sudo dnf install -y libX11-devel`
- Arch: included in `xorg-server`

**Wayland sessions** bypass the X11 library entirely and shell out to `wl-paste` from the `wl-clipboard` package — install it for paste-to-upload to work:

- Debian/Ubuntu: `sudo apt-get install -y wl-clipboard`
- Fedora/RHEL: `sudo dnf install -y wl-clipboard`
- Arch: `sudo pacman -S wl-clipboard`

slk auto-detects the session via `WAYLAND_DISPLAY` at startup. On headless Linux (or when neither dependency is met), slk runs but `Ctrl+V` smart-paste is disabled.

```bash
git clone https://github.com/agustif/slk.git
cd slk
make build       # binary at bin/slk
```

## Nix

In-tree `flake.nix` builds this source (`version = "0.0.0"`). Not a published nixpkgs package.

```bash
nix build
./result/bin/slk
```

## Windows

Clone and build (this fork has no GitHub release zip):

```powershell
git clone https://github.com/agustif/slk.git
cd slk
go build -ldflags="-s -w" -trimpath -o slk.exe ./cmd/slk
```

Add `slk.exe` to your `PATH`.

## Next steps

After installation, head to **[[Setup]]** to add your first workspace.
