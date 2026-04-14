import type { Repo } from "../types";

type SidebarProps = {
  repos: Repo[];
  selectedId: string;
  onSelect: (id: string) => void;
};

export default function Sidebar({ repos, selectedId, onSelect }: SidebarProps) {
  return (
    <aside className="sidebar">
      {/* Invisible drag region at the top — sits behind macOS traffic lights */}
      <div className="sidebar-drag" data-tauri-drag-region />

      <div className="sidebar-scroll">
        {repos.map((repo) => (
          <div key={repo.name}>
            <div className="repo-label">{repo.name}/</div>
            {repo.workspaces.map((ws) => {
              const active = ws.id === selectedId;
              return (
                <button
                  key={ws.id}
                  type="button"
                  className={`ws-row${active ? " active" : ""}`}
                  onClick={() => onSelect(ws.id)}
                >
                  <span className={`ws-dot ${ws.status}`} />
                  <span className="ws-name">{ws.name}</span>
                </button>
              );
            })}
          </div>
        ))}
      </div>

      <div className="sidebar-footer">
        <button type="button" className="new-ws-btn">
          <span>New workspace</span>
          <kbd>⌘N</kbd>
        </button>
      </div>
    </aside>
  );
}
