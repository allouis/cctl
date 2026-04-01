package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/db"
	"github.com/allouis/cctl/hook"
	"github.com/allouis/cctl/session"
)

func Hook(cfg *config.Config, store *db.DB) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	input, err := hook.ParseInput(data)
	if err != nil {
		return fmt.Errorf("parse input: %w", err)
	}
	if input == nil {
		return nil // empty session id, nothing to do
	}

	result := hook.Process(input)
	if result == nil {
		return nil // unknown event, skip
	}

	event := hook.ToEvent(input, result)
	if err := store.InsertEvent(event); err != nil {
		return fmt.Errorf("store event: %w", err)
	}

	// Wake the server's SSE hub so browsers update immediately.
	session.SignalNotify(session.NotifySocketPath(cfg.Dir))

	return nil
}
