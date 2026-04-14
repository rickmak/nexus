import { useMemo, useState } from "react";
import BottomPanel, { type BottomTab } from "./components/BottomPanel";
import Sidebar from "./components/Sidebar";
import TerminalPane from "./components/TerminalPane";
import TopBar from "./components/TopBar";
import { REPOS } from "./data/repos";

function findWorkspace(id: string) {
  for (const repo of REPOS) {
    const ws = repo.workspaces.find((w) => w.id === id);
    if (ws) return ws;
  }
  return REPOS[0].workspaces[0];
}

export default function App() {
  const [selectedWs, setSelectedWs] = useState("ws-1");
  const [activeTab, setActiveTab] = useState<BottomTab>("snapshots");
  const workspace = useMemo(() => findWorkspace(selectedWs), [selectedWs]);

  return (
    <div className="app">
      <Sidebar repos={REPOS} selectedId={selectedWs} onSelect={setSelectedWs} />
      <div className="main-pane">
        <TopBar workspace={workspace} />
        <div className="terminal-wrap">
          <TerminalPane workspaceId={workspace.id} />
        </div>
        <BottomPanel
          workspace={workspace}
          activeTab={activeTab}
          onTabChange={setActiveTab}
        />
      </div>
    </div>
  );
}
