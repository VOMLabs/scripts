# scripty

TUI script runner — navigate, compile, and run scripts interactively.

## Build

```
make
```
or:
```
go build -o scripty ./cmd/scripty/
```

## Install

```
make install
```
or:
```
sudo ./scripty install
```
or run `./scripty` and answer Y when prompted.

Installs to `~/.local/bin/scripty` by default.

## Usage

Navigate the script list with arrow keys (or `j`/`k`).  
Press **Enter** to bring up an argument prompt, then **Enter** again to run.  
Press **s** to toggle the sidebar (fullscreen output while scripts run).  
Press **r** to toggle OpenRouter mode (sends output to the API when a script finishes).  
Press **q** or **Ctrl+C** to cancel all running scripts and quit.  
Press **g** to jump to the top of output, **G** for the bottom.  
Press **PageUp** / **PageDown** to scroll faster.

Multiple scripts can run at the same time. Press **Enter** on another script while one is already running to start it in parallel. Output is merged with colored `[scriptname]` labels.

### File types detected

`.sh` `.bash` `.zsh` `.fish` `.py` `.pyw` `.js` `.mjs` `.cjs` `.rb` `.pl` `.pm` `.lua` `.ts` `.tsx` `.go` `.c` `.cc` `.cpp` `.cxx` `.rs`

Also: any file with a shebang (`#!`) or executable bit.  
Compilable files (`.c` `.cc` `.cpp` `.cxx` `.go` `.rs`) are compiled first, then run.

### Running without the TUI

```
scripty --notui script.py
scripty --notui program.c
scripty --notui txt2md.py README.txt
```

Any extra arguments after the filename are passed to the script.

Passing a directory argument scans that directory and its subdirectories for scripts:

```
scripty ~/my-scripts
```

## API Key

scripty needs an [OpenRouter](https://openrouter.ai) API key for the txt2md feature and the built-in OpenRouter mode. On first run you will be prompted to enter one. The key is stored at `~/.config/scripty/or/key.secret` (permissions 0600). You can also set the `OPENROUTER_API_KEY` environment variable.

## Scripts included

### `scripts/scripts/fix-path.py`

Rebuilds `PATH` from a clean set of known-good directories. Removes duplicates and unwanted SDK paths.

### `scripts/scripts/txt2md.py`

Converts plain text to Markdown using the OpenRouter API. Reads from stdin or a file argument. Gets the API key from `~/.config/scripty/or/key.secret` or the `OPENROUTER_API_KEY` environment variable.

Both scripts are detected and runnable from inside scripty.

## Key bindings summary

| Key | Action |
|---|---|
| `j`/`k` or `up`/`down` | Navigate script list / scroll output |
| **Enter** | Prompt for arguments then run the selected script |
| `s` | Toggle sidebar |
| `r` | Toggle OpenRouter mode |
| `q` / **Ctrl+C** | Quit (cancels all running scripts) |
| `g` | Jump to top of output |
| `G` | Jump to bottom of output |
| **PgUp** / **PgDown** | Scroll half a page |
| **Esc** | Cancel argument prompt |
