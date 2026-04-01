package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runner wraps tmux operations. Implementations can be real or mock.
type Runner interface {
	HasSession(name string) bool
	NewSession(session, window, command string) error
	NewWindow(session, window, command, dir string) (string, error)
	ListWindows(session string) ([]Window, error)
	ActiveWindowIDs(session string) map[string]bool
	SelectWindow(session, target string) error
	SendKeys(session, target, keys string) error
	CapturePane(session, target string) (string, error)
	KillWindow(session, target string) error
	KillSession(session string) error
}

type Window struct {
	ID     string // tmux window ID (@N)
	Index  string
	Name   string
	Active bool
}

// RealRunner executes actual tmux commands.
type RealRunner struct{}

func (r *RealRunner) HasSession(name string) bool {
	err := exec.Command("tmux", "has-session", "-t", name).Run()
	return err == nil
}

func (r *RealRunner) NewSession(session, window, command string) error {
	args := []string{"new-session", "-d", "-s", session, "-n", window}
	if command != "" {
		args = append(args, command)
	}
	return exec.Command("tmux", args...).Run()
}

func (r *RealRunner) NewWindow(session, window, command, dir string) (string, error) {
	args := []string{"new-window", "-P", "-F", "#{window_id}", "-t", session, "-n", window}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	if command != "" {
		args = append(args, command)
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *RealRunner) ListWindows(session string) ([]Window, error) {
	out, err := exec.Command("tmux", "list-windows", "-t", session,
		"-F", "#{window_id}:#{window_index}:#{window_name}:#{window_active}").Output()
	if err != nil {
		return nil, fmt.Errorf("list windows: %w", err)
	}

	var windows []Window
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		windows = append(windows, Window{
			ID:     parts[0],
			Index:  parts[1],
			Name:   parts[2],
			Active: parts[3] == "1",
		})
	}
	return windows, nil
}

// ActiveWindowIDs returns the set of window IDs that have a client
// actively viewing them (i.e., a human is attached and looking at that window).
func (r *RealRunner) ActiveWindowIDs(session string) map[string]bool {
	out, err := exec.Command("tmux", "list-clients", "-t", session,
		"-F", "#{window_id}").Output()
	if err != nil {
		return nil
	}
	ids := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			ids[line] = true
		}
	}
	return ids
}

func (r *RealRunner) SelectWindow(session, target string) error {
	return exec.Command("tmux", "select-window", "-t", session+":"+target).Run()
}

func (r *RealRunner) SendKeys(session, target, keys string) error {
	t := session + ":" + target

	// If the pane is in copy mode (user scrolled up), exit it first
	// so keystrokes reach the running program.
	if out, err := exec.Command("tmux", "display-message", "-t", t, "-p", "#{pane_in_mode}").Output(); err == nil {
		if strings.TrimSpace(string(out)) == "1" {
			exec.Command("tmux", "send-keys", "-t", t, "-X", "cancel").Run()
		}
	}

	// Type the text and press Enter twice: once to complete the input line,
	// once to submit it (Claude Code's TUI needs double-Enter to confirm)
	if err := exec.Command("tmux", "send-keys", "-t", t, keys, "Enter").Run(); err != nil {
		return err
	}
	return exec.Command("tmux", "send-keys", "-t", t, "Enter").Run()
}

func (r *RealRunner) CapturePane(session, target string) (string, error) {
	t := session + ":" + target
	out, err := exec.Command("tmux", "capture-pane", "-t", t, "-p", "-J").Output()
	if err != nil {
		return "", fmt.Errorf("capture pane: %w", err)
	}
	return strings.TrimRight(string(out), "\n "), nil
}

func (r *RealRunner) KillWindow(session, target string) error {
	return exec.Command("tmux", "kill-window", "-t", session+":"+target).Run()
}

func (r *RealRunner) KillSession(session string) error {
	return exec.Command("tmux", "kill-session", "-t", session).Run()
}
