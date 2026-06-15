# Glory

A WW1 terminal idle game. Built to fill the dead time while your AI coding agent runs — press **F** and make the war thump.

## What is this?

Glory is a real-time idle game that runs in your terminal. You command an Allied artillery position on the Western Front. Munitions accumulate automatically, you spend them on upgrades, and you fire barrages to push the front line forward. The game keeps ticking while you're away and rewards you for the progress when you return.

It's designed for the thirty seconds between prompts. You don't need to think hard. You just need something to do with your hands.

## Install

**Go users (recommended):**

```sh
go install github.com/andrewhorton/glory/cmd/glory@latest
```

**Prebuilt binaries** for Linux and macOS (amd64 / arm64) are available on the [GitHub Releases page](https://github.com/andrewhorton/glory/releases). Download, unarchive, and put the `glory` binary somewhere on your `$PATH`.

## Play

```sh
glory
```

### Keybinds

| Key | Action |
|-----|--------|
| `F` or `Space` | Fire artillery offensive |
| `1` | Buy upgrade: Supply Lines (+munitions rate) |
| `2` | Buy upgrade: Rifle Issue (+army power) |
| `3` | Buy upgrade: Artillery Battery (+army power) |
| `q`, `Ctrl-C`, `Esc` | Quit |

### Headless / CI mode

```sh
glory -headless
# or: GLORY_HEADLESS=1 glory
```

Runs a short deterministic core-loop smoke demo and exits. No terminal required. Useful for CI and for verifying the binary works after install.

### Version

```sh
glory -version
```

## Saves and away progress

Your save lives at `~/.config/glory/save.json` (respects `$XDG_CONFIG_HOME`).

When you relaunch, Glory calculates how long you were away and credits you the munitions you would have earned — capped at 24 hours so a long weekend doesn't trivialize the game.

To store saves somewhere else:

```sh
GLORY_SAVE_DIR=/tmp/glory-saves glory
```

## Build from source

```sh
git clone https://github.com/andrewhorton/glory.git
cd glory
go build -o glory ./cmd/glory
```

## Run tests

```sh
go test ./...
```

## Roadmap

These features are not built yet and are not documented above:

- Multiple fronts / sectors
- Prestige / new-game-plus
- Terminal sync across sessions

## License

TBD.
