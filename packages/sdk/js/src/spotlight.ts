import {
  SpotlightExposeOptions,
  SpotlightForward,
} from './types';
import type { RPCClient } from './rpc/types';

export type TunnelHandle = SpotlightForward & {
  stop: () => Promise<boolean>;
};

export type TunnelListResult = {
  forwards: TunnelHandle[];
};

export class TunnelOperations {
  private client: RPCClient;
  private workspaceId?: string;

  constructor(client: RPCClient, defaultParams: Record<string, unknown> = {}) {
    this.client = client;
    this.workspaceId = typeof defaultParams.workspaceId === 'string' ? defaultParams.workspaceId : undefined;
  }

  async add(options: SpotlightExposeOptions): Promise<TunnelHandle> {
    const workspaceId = this.resolveWorkspaceID();
    const result = await this.client.request<{ forward: SpotlightForward }>('spotlight.expose', {
      spec: {
        workspaceId,
        service: options.service,
        remotePort: options.remotePort,
        localPort: options.localPort,
        host: options.host,
      },
    });
    return this.attachStop(result.forward);
  }

  async list(): Promise<TunnelListResult> {
    const workspaceId = this.resolveWorkspaceID();
    const result = await this.client.request<{ forwards: SpotlightForward[] }>('spotlight.list', { workspaceId });
    return {
      forwards: result.forwards.map((f) => this.attachStop(f)),
    };
  }

  async stop(id: string): Promise<boolean> {
    const result = await this.client.request<{ closed: boolean }>('spotlight.close', { id });
    return result.closed;
  }

  private resolveWorkspaceID(): string {
    if (this.workspaceId && this.workspaceId.trim() !== '') {
      return this.workspaceId;
    }
    throw new Error('workspaceId is required for tunnel operation');
  }

  private attachStop(forward: SpotlightForward): TunnelHandle {
    return {
      ...forward,
      stop: async () => this.stop(forward.id),
    };
  }
}
