package config

import (
	"os"
	"path/filepath"
)

// bridgeSource is the pi extension that bridges cctl and pi.
// It translates pi lifecycle events into cctl hook calls and opens a
// Unix socket for receiving commands (prompt, abort) from cctl.
const bridgeSource = `import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { createServer } from "node:net";
import { spawn, execFileSync } from "node:child_process";
import { unlinkSync, existsSync } from "node:fs";

export default function (pi: ExtensionAPI) {
  const cctlBin = process.env.CCTL_BIN || "cctl";
  const sessionId = process.env.CCTL_SESSION_ID;
  if (!sessionId) return;

  const socketPath = "/tmp/cctl-" + sessionId + ".sock";

  function fireHook(input: Record<string, unknown>) {
    try {
      const child = spawn(cctlBin, ["hook"], {
        stdio: ["pipe", "ignore", "ignore"],
      });
      child.on("error", () => {});
      child.stdin!.write(JSON.stringify(input));
      child.stdin!.end();
    } catch {}
  }

  function capitalize(s: string): string {
    return s.charAt(0).toUpperCase() + s.slice(1);
  }

  pi.on("session_start", (_event, ctx) => {
    fireHook({
      session_id: sessionId,
      hook_event_name: "SessionStart",
      cwd: ctx.cwd,
      transcript_path: ctx.sessionManager.getSessionFile() || "",
      source: "startup",
    });
  });

  pi.on("tool_call", (event, ctx) => {
    fireHook({
      session_id: sessionId,
      hook_event_name: "PreToolUse",
      cwd: ctx.cwd,
      transcript_path: ctx.sessionManager.getSessionFile() || "",
      tool_name: capitalize(event.toolName),
      tool_input: event.input,
    });
  });

  pi.on("tool_result", (event, ctx) => {
    fireHook({
      session_id: sessionId,
      hook_event_name: "PostToolUse",
      cwd: ctx.cwd,
      transcript_path: ctx.sessionManager.getSessionFile() || "",
      tool_name: capitalize(event.toolName),
    });
  });

  pi.on("agent_end", (_event, ctx) => {
    fireHook({
      session_id: sessionId,
      hook_event_name: "Stop",
      cwd: ctx.cwd,
      transcript_path: ctx.sessionManager.getSessionFile() || "",
    });
  });

  pi.on("session_shutdown", (_event, ctx) => {
    try {
      execFileSync(cctlBin, ["hook"], {
        input: JSON.stringify({
          session_id: sessionId,
          hook_event_name: "SessionEnd",
          cwd: ctx.cwd,
          transcript_path: ctx.sessionManager.getSessionFile() || "",
          reason: "shutdown",
        }),
        timeout: 3000,
      });
    } catch {}
    server.close();
    try { unlinkSync(socketPath); } catch {}
  });

  const server = createServer((conn) => {
    let buf = "";
    conn.on("data", (chunk) => {
      buf += chunk.toString();
      let idx: number;
      while ((idx = buf.indexOf("\n")) !== -1) {
        const line = buf.slice(0, idx);
        buf = buf.slice(idx + 1);
        if (!line) continue;
        try {
          const cmd = JSON.parse(line) as { type: string; text?: string };
          switch (cmd.type) {
            case "prompt":
              if (cmd.text) {
                pi.sendUserMessage(cmd.text, { deliverAs: "followUp" });
              }
              conn.write(JSON.stringify({ ok: true }) + "\n");
              break;
            case "abort":
              try { (pi as any).abort(); } catch {}
              conn.write(JSON.stringify({ ok: true }) + "\n");
              break;
            default:
              conn.write(JSON.stringify({ error: "unknown command" }) + "\n");
          }
        } catch {
          conn.write(JSON.stringify({ error: "invalid json" }) + "\n");
        }
      }
    });
    conn.on("error", () => {});
  });

  try {
    if (existsSync(socketPath)) unlinkSync(socketPath);
  } catch {}
  server.listen(socketPath);
}
`

// WriteBridgeExtension writes the pi-bridge.ts extension to dir.
// Returns the path to the written file.
func WriteBridgeExtension(dir string) (string, error) {
	p := filepath.Join(dir, "pi-bridge.ts")
	return p, os.WriteFile(p, []byte(bridgeSource), 0o644)
}
