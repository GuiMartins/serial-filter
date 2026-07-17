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

# multiple color rules (first match wins), foreground and background;
# reads from stdin instead of a serial port when the port name is "-"
# (useful for testing without hardware)
serial-filter -color "ERROR=white/red" -color "WARN=yellow" -color "OK=green" -

# never display DEBUG lines, and of what's left only display ERROR/WARN
serial-filter -hide "DEBUG" -show "ERROR|WARN" /dev/ttyUSB0
```

### Highlighting vs. filtering

- `-color` (and its `-filter` shorthand) only **colors** matching lines; it
  never hides anything.
- `-hide` and `-show` control **visibility**: any line matching a `-hide`
  rule is never displayed; if any `-show` rule is given, only lines matching
  at least one of them are displayed. Both can be repeated and combined with
  `-color` rules on the same or different patterns.
- The `/` search and `f` clear-filter keybindings act on top of whatever
  `-hide`/`-show` already let through — they don't override them.

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

Core features implemented: multi-rule coloring (foreground + background),
show/hide filtering rules, scrollback TUI, pause/resume, history search, and
log export. Not yet implemented: persisting rules in a config file (currently
CLI flags only), per-rule enable/disable toggle at runtime, and
reconnect-on-disconnect.
