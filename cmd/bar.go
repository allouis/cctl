package cmd

import (
	"fmt"
	"strings"

	"github.com/allouis/cctl/session"
)

func Bar(svc *session.Service) error {
	sessions, err := svc.List()
	if err != nil {
		return err
	}

	var parts []string
	for _, s := range sessions {
		if s.ExecutorState == "DONE" || s.ExecutorState == "DEAD" {
			continue
		}
		icon := stateIcon(s.ExecutorState)
		parts = append(parts, fmt.Sprintf("%s%s", icon, s.Name))
	}

	if len(parts) == 0 {
		fmt.Print("no sessions")
		return nil
	}

	fmt.Print(strings.Join(parts, " "))
	return nil
}

func stateIcon(state string) string {
	switch state {
	case "WORKING":
		return "⚙ "
	case "NEEDS_INPUT":
		return "⚠ "
	case "IDLE":
		return "✓ "
	case "DEAD":
		return "✗ "
	case "DONE":
		return "● "
	case "STARTING":
		return "… "
	default:
		return "? "
	}
}
