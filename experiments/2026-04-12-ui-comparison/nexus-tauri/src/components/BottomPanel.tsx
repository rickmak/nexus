import type { Workspace } from "../types";
import { openUrl } from "@tauri-apps/plugin-opener";

export type BottomTab = "snapshots" | "ports" | "log";

type BottomPanelProps = {
  workspace: Workspace;
  activeTab: BottomTab;
  onTabChange: (tab: BottomTab) => void;
};

async function openPortUrl(port: number) {
  const url = `http://localhost:${port}`;
  try { await openUrl(url); }
  catch { console.log("open", url); }
}

function SnapshotsView({ count }: { count: number }) {
  // Build evenly-spaced snapshot labels
  const labels = count <= 1
    ? ["now"]
    : count === 2
      ? ["prev", "now"]
      : ["3h ago", "1h ago", "22m ago", "now"].slice(-(Math.min(count, 4)));

  return (
    <div style={{ padding: "18px 24px 12px" }}>
      <div style={{ position: "relative", display: "flex", alignItems: "center" }}>
        {/* connecting line */}
        <div style={{
          position: "absolute",
          left: 4, right: 4,
          top: "50%", transform: "translateY(-50%)",
          height: 1,
          background: "var(--border-strong)",
        }} />
        <div style={{
          display: "flex",
          justifyContent: "space-between",
          width: "100%",
          position: "relative",
          zIndex: 1,
        }}>
          {labels.map((label, i) => {
            const isActive = i === labels.length - 1;
            return (
              <div key={label} style={{
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                gap: 8,
                flex: 1,
              }}>
                <div style={{
                  width: isActive ? 10 : 8,
                  height: isActive ? 10 : 8,
                  borderRadius: "50%",
                  background: isActive ? "var(--accent)" : "transparent",
                  border: isActive ? "none" : "1.5px solid var(--text-3)",
                  boxShadow: isActive ? "0 0 0 3px rgba(10,132,255,0.18)" : "none",
                  cursor: "pointer",
                  transition: "transform 100ms",
                }} />
                <span style={{
                  fontSize: 10,
                  color: isActive ? "var(--text-2)" : "var(--text-3)",
                  fontVariantNumeric: "tabular-nums",
                  letterSpacing: "0.01em",
                }}>
                  {label}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function PortsView({ workspace }: { workspace: Workspace }) {
  if (workspace.ports.length === 0) {
    return (
      <div style={{ padding: "14px 20px", fontSize: 12, color: "var(--text-3)" }}>
        No forwarded ports
      </div>
    );
  }
  return (
    <div style={{ padding: "8px 16px" }}>
      {workspace.ports.map((port) => (
        <div key={port} style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "7px 0",
          fontSize: 12,
          borderBottom: "1px solid var(--border)",
        }}>
          <span>
            <span style={{ color: "var(--text-1)", fontWeight: 600, fontFamily: "var(--mono)" }}>
              {port}
            </span>
            <span style={{ color: "var(--text-3)", margin: "0 6px" }}>→</span>
            <span style={{ color: "var(--text-2)", fontFamily: "var(--mono)" }}>
              localhost:{port}
            </span>
          </span>
          <button
            type="button"
            onClick={() => void openPortUrl(port)}
            style={{
              background: "transparent",
              border: "1px solid var(--border-strong)",
              color: "var(--text-2)",
              borderRadius: 5,
              padding: "3px 10px",
              fontSize: 11,
              cursor: "pointer",
            }}
          >
            Open ↗
          </button>
        </div>
      ))}
    </div>
  );
}

function LogView() {
  const lines = [
    { t: "14:02:01", m: "workspace ready" },
    { t: "14:02:03", m: "sync: pulling refs" },
    { t: "14:02:04", m: "dev server listening" },
    { t: "14:05:22", m: "port 3000 → guest" },
    { t: "14:07:10", m: "agent session started" },
  ];
  return (
    <div style={{
      padding: "10px 16px",
      fontFamily: "var(--mono)",
      fontSize: 11,
      lineHeight: 1.7,
      overflow: "auto",
    }}>
      {lines.map((line) => (
        <div key={line.t}>
          <span style={{ color: "var(--text-3)" }}>{line.t}</span>
          {"  "}
          <span style={{ color: "var(--text-2)" }}>{line.m}</span>
        </div>
      ))}
    </div>
  );
}

export default function BottomPanel({ workspace, activeTab, onTabChange }: BottomPanelProps) {
  const tabs: { id: BottomTab; label: string }[] = [
    { id: "snapshots", label: "Snapshots" },
    { id: "ports",     label: "Ports" },
    { id: "log",       label: "Log" },
  ];

  return (
    <section className="bottom-panel">
      <div className="bottom-tabs">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            className={`bottom-tab${activeTab === tab.id ? " active" : ""}`}
            onClick={() => onTabChange(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div className="bottom-content">
        {activeTab === "snapshots" && <SnapshotsView count={workspace.snapshots} />}
        {activeTab === "ports"     && <PortsView workspace={workspace} />}
        {activeTab === "log"       && <LogView />}
      </div>
    </section>
  );
}
