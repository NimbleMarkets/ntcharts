package main

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	fmt.Println("--- Environment ---")
	envs := []string{
		"TERM", "TERM_PROGRAM", "TMUX",
		"KITTY_WINDOW_ID", "KITTY_INSTALLATION_DIR",
		"GHOSTTY_RESOURCES_DIR",
		"WEZTERM_EXECUTABLE", "WEZTERM_PANE",
	}
	for _, env := range envs {
		fmt.Printf("  %s = %q\n", env, os.Getenv(env))
	}

	fmt.Println("\n--- Probe Test ---")
	// Make stdin raw so we can read terminal responses immediately
	fd := int(os.Stdin.Fd())
	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		fmt.Printf("Error getting termios: %v (are you in a TTY?)\n", err)
		return
	}
	raw := *termios
	// Disable echo and canonical mode
	raw.Lflag &^= unix.ECHO | unix.ICANON
	// Set minimum read to 0, timeout to 1 tenth of a second (100ms)
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 1

	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &raw); err != nil {
		fmt.Printf("Error setting raw mode: %v\n", err)
		return
	}
	defer unix.IoctlSetTermios(fd, unix.TIOCSETA, termios)

	// Probe sequence: query image placement support
	probeSeq := "\x1b_Ga=q,t=d,f=24,s=1,v=1,i=42069101;AAAA\x1b\\"
	
	// If TMUX is set, we can test wrapped vs unwrapped
	var sentSeq string
	inTmux := os.Getenv("TMUX") != ""
	if inTmux {
		// Wrap using tmux passthrough format: \x1bPtmux;\x1b<escaped>\x1b\\
		// where <escaped> is probeSeq with all ESC doubled
		escaped := ""
		for _, r := range probeSeq {
			if r == '\x1b' {
				escaped += "\x1b\x1b"
			} else {
				escaped += string(r)
			}
		}
		sentSeq = "\x1bPtmux;" + escaped + "\x1b\\"
		fmt.Printf("Running inside TMUX. Sending wrapped sequence: %q\n", sentSeq)
	} else {
		sentSeq = probeSeq
		fmt.Printf("Sending standard query sequence: %q\n", sentSeq)
	}

	// Write sequence to stdout
	os.Stdout.WriteString(sentSeq)

	// Read response from stdin with 500ms timeout
	fmt.Println("Waiting for response from terminal (500ms timeout)...")
	start := time.Now()
	var response []byte
	buf := make([]byte, 1024)
	for time.Since(start) < 500*time.Millisecond {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			response = append(response, buf[:n]...)
		}
		if err != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(response) == 0 {
		fmt.Println("Result: NO response received (timeout)")
		if inTmux {
			fmt.Println("\nTip: Make sure you have 'set -g allow-passthrough on' in your ~/.tmux.conf, and reload it.")
		}
	} else {
		fmt.Printf("Result: Received %d bytes: %q\n", len(response), string(response))
	}
}
