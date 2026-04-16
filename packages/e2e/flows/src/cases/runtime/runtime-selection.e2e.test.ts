import { WorkspaceHandle, NodeInfo } from '@nexus/sdk';
import { createGitFixture, cleanupFixture } from '../../harness/repo';
import { startSession } from '../../harness/session';
import { assertCapabilityOrSkip, onWorkspaceCreateRuntimeError } from '../../harness/assertions';
import { runtimeSelectionCaseIds } from '../test-ids';

export const CASE_TEST_IDS = runtimeSelectionCaseIds;

describe('runtime selection e2e', () => {
  jest.setTimeout(420000);
  (process.platform === 'linux' ? it : it.skip)('firecracker -> workspace created when available', async () => {
    const fixture = await createGitFixture('runtime-selection-pass');
    const session = await withTimeout(startSession({
      forceManaged: true,
    }), 'startSession(firecracker-check)');

    let ws: WorkspaceHandle | null = null;
    try {
      const { capabilities: caps } = await session.client.request<NodeInfo>('node.info', {});
      if (!assertCapabilityOrSkip(caps, 'runtime.firecracker', 'runtime.firecracker capability unavailable on this host')) {
        return;
      }

      try {
        ws = await withTimeout(session.client.workspaces.create({
          repo: fixture.repoDir,
          workspaceName: 'runtime-pass',
          agentProfile: 'default',
        }), 'workspace.create(firecracker)');
      } catch (error) {
        if (onWorkspaceCreateRuntimeError(error, 'firecracker runtime path unavailable in this environment')) {
          return;
        }
        throw error;
      }

      const rows = await session.client.workspaces.list();
      const created = rows.find((row) => row.id === ws?.id);
      expect(created?.backend).toBe('firecracker');
    } finally {
      if (ws) {
        await session.client.workspaces.remove(ws.id);
      }
      await session.stop();
      await cleanupFixture(fixture);
    }
  });
});

const RUNTIME_E2E_OP_MS = 300000;

async function withTimeout<T>(promise: Promise<T>, label: string, timeoutMs = RUNTIME_E2E_OP_MS): Promise<T> {
  return Promise.race([
    promise,
    new Promise<T>((_, reject) => {
      setTimeout(() => reject(new Error(`${label} timed out after ${String(timeoutMs)}ms`)), timeoutMs);
    }),
  ]);
}
