import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

// Per-workspace fake sessions
const SESSIONS: Record<string, string[]> = {
  "ws-1": [
    "\r\n\x1b[32m❯\x1b[0m claude --continue\r\n",
    "\x1b[90m[claude]\x1b[0m Resuming previous session...\r\n",
    "\x1b[90m[claude]\x1b[0m Analyzing codebase...\r\n",
    "\x1b[90m[claude]\x1b[0m Reading \x1b[37msrc/auth/oauth.ts\x1b[0m\r\n",
    "\x1b[90m[claude]\x1b[0m Editing \x1b[37msrc/auth/oauth.ts\x1b[0m\r\n",
    "\r\n\x1b[32m❯\x1b[0m \x1b[5m▋\x1b[0m",
  ],
  "ws-2": [
    "\r\n\x1b[32m❯\x1b[0m claude --continue\r\n",
    "\x1b[90m[claude]\x1b[0m Resuming previous session...\r\n",
    "\x1b[90m[claude]\x1b[0m Reviewing API structure...\r\n",
    "\x1b[90m[claude]\x1b[0m Refactoring \x1b[37msrc/api/v2/router.ts\x1b[0m\r\n",
    "\r\n\x1b[32m❯\x1b[0m \x1b[5m▋\x1b[0m",
  ],
  "ws-3": [
    "\r\n\x1b[32m❯\x1b[0m claude --continue\r\n",
    "\x1b[90m[claude]\x1b[0m Session active on main branch\r\n",
    "\x1b[90m[claude]\x1b[0m Watching for changes...\r\n",
    "\r\n\x1b[32m❯\x1b[0m \x1b[5m▋\x1b[0m",
  ],
};

export default function TerminalPane({ workspaceId }: { workspaceId: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef      = useRef<Terminal | null>(null);
  const fitRef       = useRef<FitAddon | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    // Tear down previous instance
    termRef.current?.dispose();

    const term = new Terminal({
      theme: {
        background:    "#161618",
        foreground:    "#e8e8ed",
        cursor:        "#e8e8ed",
        cursorAccent:  "#161618",
        selectionBackground: "rgba(255,255,255,0.15)",
        black:         "#1c1c1f",
        red:           "#ff453a",
        green:         "#30d158",
        yellow:        "#ffd60a",
        blue:          "#0a84ff",
        magenta:       "#bf5af2",
        cyan:          "#5ac8fa",
        white:         "#e8e8ed",
        brightBlack:   "#636366",
        brightWhite:   "#ffffff",
      },
      fontFamily: '"SF Mono", "Menlo", "Monaco", monospace',
      fontSize:   13,
      lineHeight: 1.6,
      letterSpacing: 0.3,
      cursorBlink: true,
      cursorStyle: "bar",
      scrollback: 1000,
      allowTransparency: true,
      fontWeight: "normal",
      fontWeightBold: "bold",
    });

    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(containerRef.current);
    fit.fit();

    termRef.current = term;
    fitRef.current  = fit;

    // Write session with slight streaming delay for realism
    const lines = SESSIONS[workspaceId] ?? SESSIONS["ws-1"];
    let i = 0;
    const write = () => {
      if (i < lines.length) {
        term.write(lines[i++]);
        if (i < lines.length) setTimeout(write, 60);
      }
    };
    setTimeout(write, 120);

    const ro = new ResizeObserver(() => fit.fit());
    ro.observe(containerRef.current);
    return () => { ro.disconnect(); term.dispose(); };
  }, [workspaceId]);

  return (
    <div
      ref={containerRef}
      style={{
        position: "absolute",
        inset: 0,
        padding: "8px 4px 0",
        background: "#161618",
        overflow: "hidden",
      }}
    />
  );
}
