package cmd

import (
	"fmt"
	"strconv"

	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/server"
	"github.com/allouis/cctl/session"
)

func Serve(cfg *config.Config, svc *session.Service, args []string) error {
	port := cfg.Port
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					port = n
				}
				i++
			}
		default:
			if n, err := strconv.Atoi(args[i]); err == nil {
				port = n
			}
		}
	}

	notify, cleanup, err := session.ListenNotify(session.NotifySocketPath(cfg.Dir))
	if err != nil {
		return fmt.Errorf("listen notify: %w", err)
	}
	defer cleanup()

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting dashboard at http://localhost%s\n", addr)

	srv := server.New(svc, cfg, notify)
	return srv.ListenAndServe(addr)
}
