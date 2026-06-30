# fix-mic

Fixes Logitech PRO X headset microphone issues on Linux (PipeWire/PulseAudio).

## What it does

- Detects the Logitech PRO X audio source
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
go build -o fix-mic .
```

## Usage

```
./fix-mic
```

Press `q`, `esc`, or `Ctrl+C` to quit while running.

## Requirements

- `pactl`, `parec`, and `systemctl --user` (part of PipeWire/PulseAudio)
- `discord` and `vesktop` binaries on `PATH` (optional — for automatic restart)
