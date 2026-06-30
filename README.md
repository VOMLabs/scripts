# headset-fixer

Fixes USB headset microphone issues on Linux (PipeWire/PulseAudio).

Supports any USB headset — picks your mic from a list, then applies fixes.

## What it does

- Detects all USB headset microphones and lets you pick one
- Unloads echo-cancel modules that interfere with the mic
- Disables echo-cancel config files
- Sets the correct default source and unmutes
- Toggles the mute LED via `/dev/input` if stuck on
- Restarts PipeWire-Pulse if changes were made
- Restarts Discord and Vesktop to re-establish audio connections
- Verifies audio is working

## Build

```
meson setup build
meson compile -C build
```

Or directly:

```
go build -o headset-fixer .
```

## Usage

```
./headset-fixer
```

Arrow keys to pick your mic, Enter to start, `q`/`esc`/`Ctrl+C` to quit.

## Requirements

- `pactl`, `parec`, and `systemctl --user` (part of PipeWire/PulseAudio)
- `discord` and `vesktop` binaries on `PATH` (optional — for automatic restart)
