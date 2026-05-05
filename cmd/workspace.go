package cmd

import (
	"fmt"

	"github.com/allouis/cctl/session"
)

func Workspace(svc *session.Service, args []string) error {
	if err := checkHelp(args, "usage: cctl workspace <prune>"); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cctl workspace <prune>")
	}
	switch args[0] {
	case "prune":
		return workspacePrune(svc)
	default:
		return fmt.Errorf("unknown workspace subcommand: %s", args[0])
	}
}

func workspacePrune(svc *session.Service) error {
	res, err := svc.PruneWorkspaces()
	if err != nil {
		return err
	}
	if len(res.Pruned) == 0 && len(res.Retained) == 0 && len(res.Orphans) == 0 {
		fmt.Println("No workspaces to prune.")
		return nil
	}
	if len(res.Pruned) > 0 {
		fmt.Printf("Pruned %d workspace(s):\n", len(res.Pruned))
		for _, name := range res.Pruned {
			fmt.Printf("  - %s\n", name)
		}
	}
	if len(res.Orphans) > 0 {
		fmt.Printf("Removed %d orphaned workspace(s):\n", len(res.Orphans))
		for _, name := range res.Orphans {
			fmt.Printf("  - %s\n", name)
		}
	}
	if len(res.Retained) > 0 {
		fmt.Printf("Retained %d workspace(s) with unlanded work:\n", len(res.Retained))
		for _, name := range res.Retained {
			fmt.Printf("  - %s\n", name)
		}
	}
	return nil
}
