<script lang="ts">
  export let workspace: {
    id: string;
    name: string;
    branch: string;
    status: string;
    ports: number[];
    snapshotCount: number;
  } | null = null;

  let tab: 'snapshots' | 'ports' | 'log' = 'snapshots';

  const snapshotLabels = ['2h ago', '45m ago', '12m ago', 'now'];

  const logLines = [
    { t: '14:02:01', m: 'Workspace ready on branch feat/oauth' },
    { t: '14:02:04', m: 'Port 3000 forwarded to localhost:3000' },
    { t: '14:03:22', m: 'Snapshot created (manual)' },
    { t: '14:05:11', m: 'SSH session connected from 10.0.1.4' },
  ];
</script>

<div class="bottom">
  <div class="tabs" role="tablist">
    <button
      type="button"
      class="tab"
      class:active={tab === 'snapshots'}
      on:click={() => (tab = 'snapshots')}
      role="tab"
      aria-selected={tab === 'snapshots'}
    >
      Snapshots
    </button>
    <button
      type="button"
      class="tab"
      class:active={tab === 'ports'}
      on:click={() => (tab = 'ports')}
      role="tab"
      aria-selected={tab === 'ports'}
    >
      Ports
    </button>
    <button
      type="button"
      class="tab"
      class:active={tab === 'log'}
      on:click={() => (tab = 'log')}
      role="tab"
      aria-selected={tab === 'log'}
    >
      Log
    </button>
  </div>

  <div class="panel" role="tabpanel">
    {#if tab === 'snapshots'}
      <div class="timeline">
        <div class="timeline-line" />
        {#each snapshotLabels as label, i}
          <div class="timeline-node">
            <div
              class="dot"
              class:filled={i === snapshotLabels.length - 1}
              class:outline={i !== snapshotLabels.length - 1}
            />
            <span class="timeline-label">{label}</span>
          </div>
        {/each}
      </div>
    {:else if tab === 'ports'}
      <ul class="ports">
        {#if workspace && workspace.ports.length}
          {#each workspace.ports as p}
            <li class="port-row">
              <span class="port-map">{p} → localhost:{p}</span>
              <button type="button" class="open-link">Open ↗</button>
            </li>
          {/each}
        {:else}
          <li class="empty">No forwarded ports</li>
        {/if}
      </ul>
    {:else}
      <ul class="log">
        {#each logLines as line}
          <li><span class="ts">{line.t}</span> {line.m}</li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .bottom {
    flex-shrink: 0;
    border-top: 1px solid var(--border);
    background: var(--bg-main);
    min-height: 160px;
    display: flex;
    flex-direction: column;
  }
  .tabs {
    display: flex;
    gap: 0;
    border-bottom: 1px solid var(--border);
    padding: 0 12px;
  }
  .tab {
    padding: 8px 12px;
    font-size: 12px;
    color: var(--text-2);
    background: transparent;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
  }
  .tab:hover {
    color: var(--text-1);
  }
  .tab.active {
    color: var(--text-1);
    border-bottom-color: var(--accent);
  }
  .panel {
    flex: 1;
    min-height: 0;
    padding: 12px 16px;
    overflow: auto;
  }
  .timeline {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    position: relative;
    padding-top: 8px;
    max-width: 520px;
  }
  .timeline-line {
    position: absolute;
    left: 12px;
    right: 12px;
    top: 15px;
    height: 1px;
    background: var(--border-strong);
    pointer-events: none;
  }
  .timeline-node {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }
  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .dot.outline {
    background: var(--bg-main);
    border: 2px solid var(--text-3);
  }
  .dot.filled {
    background: var(--accent);
    border: 2px solid var(--accent);
  }
  .timeline-label {
    font-size: 11px;
    color: var(--text-2);
    white-space: nowrap;
  }
  .ports {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .port-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 12px;
    color: var(--text-1);
  }
  .port-map {
    font-family: var(--mono);
    font-size: 12px;
  }
  .open-link {
    font-size: 12px;
    color: var(--text-2);
    padding: 4px 10px;
    border: 1px solid var(--border-strong);
    border-radius: 4px;
  }
  .open-link:hover {
    color: var(--text-1);
    background: var(--bg-hover);
  }
  .empty {
    font-size: 12px;
    color: var(--text-3);
  }
  .log {
    font-family: var(--mono);
    font-size: 11px;
    line-height: 1.6;
    color: var(--text-2);
  }
  .log .ts {
    color: var(--text-3);
    margin-right: 8px;
  }
</style>
