<script lang="ts">
  import { onMount } from 'svelte';
  import Terminal from './lib/Terminal.svelte';
  import BottomPanel from './lib/BottomPanel.svelte';

  type WorkspaceRow = {
    id: string;
    name: string;
    branch: string;
    status: string;
    ports: number[];
    snapshotCount: number;
  };

  type Repo = { name: string; workspaces: WorkspaceRow[] };

  const REPOS: Repo[] = [
    {
      name: 'nexus',
      workspaces: [
        {
          id: 'ws-1',
          name: 'auth-feature',
          branch: 'feat/oauth',
          status: 'running',
          ports: [3000, 8080],
          snapshotCount: 4,
        },
        {
          id: 'ws-2',
          name: 'api-refactor',
          branch: 'refactor/v2',
          status: 'paused',
          ports: [],
          snapshotCount: 2,
        },
      ],
    },
    {
      name: 'magic',
      workspaces: [
        {
          id: 'ws-3',
          name: 'main',
          branch: 'main',
          status: 'running',
          ports: [4000],
          snapshotCount: 7,
        },
      ],
    },
  ];

  let repos: Repo[] = REPOS;
  let selectedId = 'ws-1';

  let GetWorkspaces: (() => Promise<Repo[]>) | undefined;
  let WorkspaceAction: ((id: string, action: string) => Promise<void>) | undefined;

  function findWorkspace(id: string): WorkspaceRow | null {
    for (const r of repos) {
      const w = r.workspaces.find((x) => x.id === id);
      if (w) return w;
    }
    return null;
  }

  $: selected = findWorkspace(selectedId);

  function statusDotClass(status: string): string {
    if (status === 'running') return 'dot-green';
    if (status === 'paused') return 'dot-orange';
    return 'dot-red';
  }

  function statusLabel(status: string): string {
    if (status === 'running') return 'Running';
    if (status === 'paused') return 'Paused';
    return 'Stopped';
  }

  async function loadFromBackend() {
    try {
      if (typeof GetWorkspaces === 'function') {
        repos = await GetWorkspaces();
        if (!findWorkspace(selectedId) && repos.length) {
          const first = repos[0].workspaces[0];
          if (first) selectedId = first.id;
        }
      }
    } catch {
      repos = REPOS;
    }
  }

  async function handleStopStart() {
    const w = selected;
    if (!w) return;
    const action = w.status === 'running' ? 'stop' : 'start';
    if (typeof WorkspaceAction === 'function') {
      await WorkspaceAction(w.id, action);
    } else {
      await new Promise((r) => setTimeout(r, 200));
    }
    const defaults = REPOS.flatMap((r) => r.workspaces).find((x) => x.id === w.id);
    const restorePorts = defaults?.ports ?? [];
    repos = repos.map((r) => ({
      ...r,
      workspaces: r.workspaces.map((x) => {
        if (x.id !== w.id) return x;
        if (action === 'stop') return { ...x, status: 'paused', ports: [] };
        return { ...x, status: 'running', ports: restorePorts };
      }),
    }));
  }

  function onKeyDown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'n') {
      e.preventDefault();
      alert('New workspace');
    }
  }

  onMount(() => {
    import('./wailsjs/go/main/App')
      .then((m) => {
        GetWorkspaces = m.GetWorkspaces;
        WorkspaceAction = m.WorkspaceAction;
      })
      .catch(() => {})
      .finally(() => loadFromBackend());

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  });
</script>

<div class="app">
  <aside class="sidebar">
    <div class="tree">
      {#each repos as repo (repo.name)}
        <div class="repo-name">{repo.name}/</div>
        <ul class="ws-list">
          {#each repo.workspaces as ws (ws.id)}
            <li>
              <button
                type="button"
                class="ws-row"
                class:selected={selectedId === ws.id}
                on:click={() => (selectedId = ws.id)}
              >
                <span class="status-dot {statusDotClass(ws.status)}" aria-hidden="true" />
                <span class="ws-name">{ws.name}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/each}
    </div>
    <button type="button" class="new-ws" on:click={() => alert('New workspace')}>
      ⌘N New workspace
    </button>
  </aside>

  <section class="main">
    {#if selected}
      <header class="topbar">
        <div class="topbar-left">
          <span class="ws-title">{selected.name}</span>
          <span class="sep">·</span>
          <span class="branch">{selected.branch}</span>
        </div>
        <div class="topbar-right">
          <span class="chip">
            <span class="status-dot {statusDotClass(selected.status)}" aria-hidden="true" />
            {statusLabel(selected.status)}
          </span>
          <button type="button" class="btn-ghost">Open in SSH</button>
          <button type="button" class="btn-ghost" on:click={handleStopStart}>
            {selected.status === 'running' ? 'Stop' : 'Start'}
          </button>
        </div>
      </header>
    {:else}
      <header class="topbar">
        <div class="topbar-left"><span class="ws-title">No workspace</span></div>
      </header>
    {/if}

    <div class="main-body">
      <Terminal />
      <BottomPanel workspace={selected} />
    </div>
  </section>
</div>

<style>
  .app {
    display: flex;
    height: 100vh;
    min-height: 0;
    overflow: hidden;
    background: var(--bg-app);
  }
  .sidebar {
    width: 220px;
    flex-shrink: 0;
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .tree {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 8px 0 12px;
  }
  .repo-name {
    font-size: 10px;
    font-variant: small-caps;
    letter-spacing: 0.08em;
    color: var(--text-3);
    padding: 8px 12px 4px;
    user-select: none;
    pointer-events: none;
  }
  .ws-list {
    margin-bottom: 4px;
  }
  .ws-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    text-align: left;
    padding: 6px 12px 6px 24px;
    font-size: 13px;
    color: var(--text-1);
    background: transparent;
    border: none;
    border-left: 2px solid transparent;
    cursor: pointer;
  }
  .ws-row:hover {
    background: var(--bg-hover);
  }
  .ws-row.selected {
    background: var(--bg-selected);
    border-left-color: var(--accent);
  }
  .ws-name {
    min-width: 0;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .dot-green {
    background: var(--green);
  }
  .dot-orange {
    background: var(--orange);
  }
  .dot-red {
    background: var(--red);
  }
  .new-ws {
    flex-shrink: 0;
    padding: 10px 12px;
    font-size: 12px;
    color: var(--text-3);
    text-align: left;
    background: transparent;
    border-top: 1px solid var(--border);
  }
  .new-ws:hover {
    color: var(--text-2);
  }
  .main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg-main);
  }
  .topbar {
    flex-shrink: 0;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-main);
  }
  .topbar-left {
    display: flex;
    align-items: baseline;
    gap: 6px;
    min-width: 0;
  }
  .ws-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-1);
  }
  .sep {
    color: var(--text-3);
    font-size: 12px;
  }
  .branch {
    font-family: var(--mono);
    font-size: 12px;
    color: var(--text-2);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .topbar-right {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--text-2);
  }
  .btn-ghost {
    font-size: 12px;
    padding: 4px 10px;
    color: var(--text-1);
    border: 1px solid var(--border-strong);
    border-radius: 4px;
    background: transparent;
  }
  .btn-ghost:hover {
    background: var(--bg-hover);
  }
  .main-body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
</style>
