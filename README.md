# serial-filter

A cross-platform (Windows, Linux, macOS) serial console for viewing logs from
embedded hardware, similar in spirit to `minicom` but focused on log viewing:
connect to a serial port, stream its output in a scrollable TUI, colorize
lines by rule, search the history, and export the captured log.

## Usage

```sh
# list available serial ports
serial-filter -list

# connect, highlighting ERROR lines in yellow
serial-filter -baud 115200 -filter "ERROR" /dev/ttyUSB0

# start with the view already filtered to ERROR/WARN lines only
serial-filter -filter "ERROR|WARN" -only-matching /dev/ttyUSB0

# multiple color rules (first match wins); reads from stdin instead of
# a serial port when the port name is "-" (useful for testing without hardware)
serial-filter -color "ERROR=red" -color "WARN=yellow" -color "OK=green" -
```

## Keybindings

| Key | Action |
| --- | --- |
| `q` / `Ctrl+C` | quit |
| `p` | pause / resume the live view (reading continues in the background) |
| `/` | search the full history (regex, falls back to plain substring) |
| `f` | clear the active filter |
| `e` | export the full captured log to `serial-filter-export-<timestamp>.log` |
| arrows / `pgup` / `pgdn` | scroll |

## Status

Core features implemented: multi-rule coloring, scrollback TUI, pause/resume,
history search, and log export. Not yet implemented: persisting color rules
in a config file (currently CLI flags only) and reconnect-on-disconnect.
