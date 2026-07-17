// Command serial-filter is a cross-platform serial console with regex
// highlighting, similar in spirit to minicom but focused on log viewing.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/fatih/color"
	"go.bug.st/serial"
)

func main() {
	var (
		listPorts = flag.Bool("list", false, "list available serial ports and exit")
		baud      = flag.Int("baud", 115200, "baud rate")
		filter    = flag.String("filter", "", "regex to highlight in the output")
	)
	flag.Parse()

	if *listPorts {
		ports, err := serial.GetPortsList()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error listing ports:", err)
			os.Exit(1)
		}
		if len(ports) == 0 {
			fmt.Println("no serial ports found")
			return
		}
		for _, p := range ports {
			fmt.Println(p)
		}
		return
	}

	portName := flag.Arg(0)
	if portName == "" {
		fmt.Fprintln(os.Stderr, "usage: serial-filter [-baud N] [-filter regex] <port>")
		os.Exit(1)
	}

	var highlight *regexp.Regexp
	if *filter != "" {
		re, err := regexp.Compile(*filter)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid filter regex:", err)
			os.Exit(1)
		}
		highlight = re
	}

	mode := &serial.Mode{BaudRate: *baud}
	port, err := serial.Open(portName, mode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening port:", err)
		os.Exit(1)
	}
	defer port.Close()

	scanner := bufio.NewScanner(port)
	for scanner.Scan() {
		line := scanner.Text()
		printLine(line, highlight)
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, "read error:", err)
		os.Exit(1)
	}
}

func printLine(line string, highlight *regexp.Regexp) {
	if highlight == nil {
		fmt.Println(line)
		return
	}
	if highlight.MatchString(line) {
		color.New(color.FgYellow, color.Bold).Println(line)
		return
	}
	fmt.Println(line)
}
