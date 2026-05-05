package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/allouis/cctl/db"
	"github.com/allouis/cctl/session"
	"github.com/allouis/cctl/tmux"
)

func New(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl new <name> [dir] [-p prompt] [--harness claude|pi] [--safe]"); err != nil {
		return err
	}

	var prompt string
	var harness string
	var safe bool
	var positional []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p", "--prompt":
			if i+1 < len(args) {
				prompt = args[i+1]
				i++
			}
		case "--harness":
			if i+1 < len(args) {
				harness = args[i+1]
				i++
			}
		case "--safe":
			safe = true
		default:
			positional = append(positional, args[i])
		}
	}

	if len(positional) < 1 {
		return fmt.Errorf("usage: cctl new <name> [dir] [-p prompt] [--harness claude|pi] [--safe]")
	}

	name := positional[0]
	dir, _ := os.Getwd()
	if len(positional) > 1 {
		dir = positional[1]
	}

	if _, err := svc.Create(session.CreateOpts{
		Name:    name,
		Dir:     dir,
		Prompt:  prompt,
		Safe:    safe,
		Harness: harness,
	}); err != nil {
		return err
	}

	fmt.Printf("Started session '%s'\n", name)
	return nil
}

func Resume(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl resume <name>"); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cctl resume <name>")
	}

	if err := svc.Resume(args[0]); err != nil {
		return err
	}

	fmt.Printf("Resumed session '%s'\n", args[0])
	return nil
}

func List(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl ls [-a|--all]"); err != nil {
		return err
	}
	all := false
	for _, a := range args {
		if a == "--all" || a == "-a" {
			all = true
		}
	}

	sessions, err := svc.List()
	if err != nil {
		return err
	}

	if !all {
		var active []db.Session
		for _, s := range sessions {
			if s.ExecutorState != "DONE" && s.ExecutorState != "DEAD" {
				active = append(active, s)
			}
		}
		sessions = active
	}

	if len(sessions) == 0 {
		fmt.Println("No active Claude Code sessions")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tDETAIL\tTOOL")
	for _, s := range sessions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.ExecutorState, s.ExecutorDetail, s.Tool)
	}
	w.Flush()
	return nil
}

func Peek(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl peek <name>"); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cctl peek <name>")
	}

	s, err := svc.Get(args[0])
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("session '%s' not found", args[0])
	}

	fmt.Printf("Session: %s\n", s.Name)
	fmt.Printf("State:   %s\n", s.ExecutorState)
	fmt.Printf("Detail:  %s\n", s.ExecutorDetail)
	fmt.Println()
	if s.Preview != "" {
		fmt.Println(s.Preview)
	} else {
		fmt.Println("(no preview available)")
	}
	return nil
}

func Log(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl log <name> [n]"); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cctl log <name> [n]")
	}

	name := args[0]
	limit := 20
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil {
			limit = n
		}
	}

	entries, err := svc.Transcript(name, limit)
	if err != nil {
		return err
	}

	for _, e := range entries {
		prefix := ">"
		if e.Role == "assistant" {
			prefix = "◆"
		}
		fmt.Printf("%s %s\n\n", prefix, e.Text)
	}
	return nil
}

func Go(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl go <name|number>"); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cctl go <name|number>")
	}
	if err := svc.Focus(args[0]); err != nil {
		return fmt.Errorf("window '%s' not found", args[0])
	}
	return nil
}

func Send(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl send <name> <text>"); err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: cctl send <name> <text>")
	}

	name := args[0]
	text := strings.Join(args[1:], " ")

	result, err := svc.Send(name, text)
	if err != nil {
		return fmt.Errorf("send to '%s': %w", name, err)
	}
	if result.Confirmed {
		fmt.Println("Message delivered")
	} else {
		fmt.Println("Message sent (delivery not confirmed)")
	}
	return nil
}

func Kill(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl kill <name>"); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cctl kill <name>")
	}

	if err := svc.Kill(args[0]); err != nil {
		return err
	}
	fmt.Printf("Killed session '%s'\n", args[0])
	return nil
}

func Delete(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl delete <name>"); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cctl delete <name>")
	}

	if err := svc.Delete(args[0]); err != nil {
		return err
	}
	fmt.Printf("Deleted session '%s'\n", args[0])
	return nil
}

func Attach(runner tmux.Runner, session string, args []string) error {
	if err := checkHelp(args, "usage: cctl attach [name]"); err != nil {
		return err
	}
	if len(args) > 0 {
		runner.SelectWindow(session, args[0])
	}

	tmuxPath, err := findTmux()
	if err != nil {
		return err
	}
	return syscallExec(tmuxPath, []string{"tmux", "attach-session", "-t", session}, os.Environ())
}

func findTmux() (string, error) {
	path, err := findExecutable("tmux")
	if err != nil {
		return "", fmt.Errorf("tmux not found in PATH")
	}
	return path, nil
}
