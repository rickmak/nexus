import type { Workspace } from "../types";

type TopBarProps = { workspace: Workspace };

// Feather-style share icon (matches macOS share symbol)
function ShareIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 12v8a2 2 0 002 2h12a2 2 0 002-2v-8" />
      <polyline points="16 6 12 2 8 6" />
      <line x1="12" y1="2" x2="12" y2="15" />
    </svg>
  );
}

// Three dots (ellipsis) icon
function MoreIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor">
      <circle cx="5"  cy="12" r="1.5" />
      <circle cx="12" cy="12" r="1.5" />
      <circle cx="19" cy="12" r="1.5" />
    </svg>
  );
}

export default function TopBar({ workspace }: TopBarProps) {
  const statusLabel = workspace.status === "running" ? "Running" : "Paused";

  return (
    <header className="topbar" data-tauri-drag-region>
      <div className="topbar-left">
        <span className="topbar-name">{workspace.name}</span>
        <span className="topbar-sep">·</span>
        <span className="topbar-branch">{workspace.branch}</span>
      </div>

      <div className="topbar-right">
        <span className="status-pill">{statusLabel}</span>
        <button type="button" className="icon-btn" title="Share">
          <ShareIcon />
        </button>
        <button type="button" className="icon-btn" title="More">
          <MoreIcon />
        </button>
      </div>
    </header>
  );
}
