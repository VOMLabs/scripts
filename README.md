# fix-mic

Fixes Logitech PRO X headset microphone issues on Linux (PipeWire/PulseAudio).

## What it does

- Detects the Logitech PRO X audio source
- Unloads echo-cancel modules that interfere with the mic
- Disables echo-cancel config files
- Sets the correct default source and unmutes
- Toggles the mute LED via `/dev/input` if stuck on
- Restarts PipeWire-Pulse if changes were made
- Verifies audio is working

## Build

```
make
```

or:

```
go build -o fix-mic .
```

## Usage

```
./fix-mic
```

Requires `pactl`, `parec`, and `systemctl --user` (part of PipeWire/PulseAudio).
