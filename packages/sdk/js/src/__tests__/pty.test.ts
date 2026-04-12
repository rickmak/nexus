import { PTYOperations } from '../pty';

describe('PTYOperations', () => {
  it('sends pty RPC requests', async () => {
    const calls: Array<{ method: string; params?: Record<string, unknown> }> = [];
    const request = async <T = unknown>(method: string, params?: Record<string, unknown>): Promise<T> => {
      calls.push({ method, params });
      let result: unknown = {};
      if (method === 'pty.open') {
        result = { sessionId: 'pty-123' };
      } else if (method === 'pty.write' || method === 'pty.resize') {
        result = { ok: true };
      } else if (method === 'pty.close') {
        result = { closed: true };
      }
      return result as T;
    };
    const onNotification = jest.fn(() => () => {});

    const pty = new PTYOperations({ request, onNotification });

    await expect(pty.open({ workspaceId: 'ws-1' })).resolves.toBe('pty-123');
    await expect(pty.write('pty-123', 'ls\n')).resolves.toBe(true);
    await expect(pty.resize('pty-123', 120, 40)).resolves.toBe(true);
    await expect(pty.close('pty-123')).resolves.toBe(true);

    expect(calls).toEqual([
      { method: 'pty.open', params: { workspaceId: 'ws-1' } },
      { method: 'pty.write', params: { sessionId: 'pty-123', data: 'ls\n' } },
      { method: 'pty.resize', params: { sessionId: 'pty-123', cols: 120, rows: 40 } },
      { method: 'pty.close', params: { sessionId: 'pty-123' } },
    ]);
  });

  it('filters malformed notification payloads', () => {
    const request = async <T = unknown>(): Promise<T> => ({} as T);
    const listeners = new Map<string, (params: unknown) => void>();
    const onNotification = jest.fn((method: string, cb: (params: unknown) => void) => {
      listeners.set(method, cb);
      return () => listeners.delete(method);
    });

    const pty = new PTYOperations({ request, onNotification });
    const onData = jest.fn();
    const onExit = jest.fn();
    pty.onData(onData);
    pty.onExit(onExit);

    listeners.get('pty.data')?.({ bad: true });
    listeners.get('pty.exit')?.({ sessionId: 'pty-1', exitCode: '1' });
    listeners.get('pty.data')?.({ sessionId: 'pty-1', data: 'ok' });
    listeners.get('pty.exit')?.({ sessionId: 'pty-1', exitCode: 1 });

    expect(onData).toHaveBeenCalledTimes(1);
    expect(onExit).toHaveBeenCalledTimes(1);
  });
});
