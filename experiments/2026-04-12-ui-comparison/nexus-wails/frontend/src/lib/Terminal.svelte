<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';

  let el: HTMLDivElement;
  let term: Terminal;
  let fit: FitAddon;
  let ro: ResizeObserver;

  const MOCK =
    '\r\n\x1b[32m❯\x1b[0m claude --continue\r\n' +
    '\x1b[90m[claude]\x1b[0m Analyzing codebase...\r\n' +
    '\x1b[90m[claude]\x1b[0m Editing src/auth/oauth.ts\r\n' +
    '\x1b[32m❯\x1b[0m _';

  export function write(text: string) {
    if (term) term.write(text);
  }

  onMount(() => {
    term = new Terminal({
      fontFamily: '"SF Mono", Monaco, "Cascadia Code", monospace',
      fontSize: 13,
      cursorBlink: true,
      theme: {
        background: '#141416',
        foreground: '#e8e8ed',
        cursor: '#0a84ff',
        cursorAccent: '#141416',
        selectionBackground: 'rgba(10, 132, 255, 0.25)',
      },
    });
    fit = new FitAddon();
    term.loadAddon(fit);
    term.open(el);
    fit.fit();
    term.write(MOCK);

    ro = new ResizeObserver(() => {
      fit.fit();
    });
    ro.observe(el);
  });

  onDestroy(() => {
    ro?.disconnect();
    term?.dispose();
  });
</script>

<div class="term-wrap" bind:this={el} />

<style>
  .term-wrap {
    flex: 1;
    min-height: 0;
    width: 100%;
    background: var(--bg-main);
  }
  .term-wrap :global(.xterm) {
    height: 100%;
    padding: 8px 12px;
  }
  .term-wrap :global(.xterm-viewport) {
    background-color: var(--bg-main) !important;
  }
</style>
