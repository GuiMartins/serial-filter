# serial-filter

A cross-platform (Windows, Linux, macOS) serial console for viewing logs from
embedded hardware, similar in spirit to `minicom` but focused on log viewing:
connect to a serial port, stream its output, and highlight lines matching a
filter.

## Usage

```sh
# list available serial ports
serial-filter -list

# connect and stream, highlighting lines that match a regex
serial-filter -baud 115200 -filter "ERROR|WARN" /dev/ttyUSB0
```

## Status

Early scaffold. Planned next steps: colorized rule sets beyond a single
filter, a full TUI (scrollback, pause/resume), and log export.
