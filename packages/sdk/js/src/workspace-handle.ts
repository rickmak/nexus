import { ExecOperations } from './exec';
import { FSOperations } from './fs';
import { TunnelOperations } from './spotlight';
import type { RPCClient } from './rpc/types';
import {
  ExecOptions,
  WorkspaceReadyCheck,
  WorkspaceReadyResult,
  WorkspaceRecord,
} from './types';

export class WorkspaceHandle {
  private client: RPCClient;
  private record: WorkspaceRecord;
  private readonly execOps: ExecOperations;
  private readonly fsOps: FSOperations;
  public readonly tunnel: TunnelOperations;

  constructor(client: RPCClient, record: WorkspaceRecord) {
    this.client = client;
    this.record = record;
    const scopedParams = { workspaceId: record.id };
    this.execOps = new ExecOperations(client, scopedParams);
    this.fsOps = new FSOperations(client, scopedParams);
    this.tunnel = new TunnelOperations(client, scopedParams);
  }

  get id(): string {
    return this.record.id;
  }

  get state(): string {
    return this.record.state;
  }

  get rootPath(): string {
    return this.record.rootPath;
  }

  async ready(
    checksOrProfile: WorkspaceReadyCheck[] | string,
    options?: { timeoutMs?: number; intervalMs?: number }
  ): Promise<WorkspaceReadyResult> {
    if (typeof checksOrProfile === 'string') {
      return this.client.request<WorkspaceReadyResult>('workspace.ready', {
        workspaceId: this.record.id,
        profile: checksOrProfile,
        ...options,
      });
    }
    return this.client.request<WorkspaceReadyResult>('workspace.ready', {
      workspaceId: this.record.id,
      checks: checksOrProfile,
      ...options,
    });
  }

  async exec(command: string, args: string[] = [], options: ExecOptions = {}) {
    return this.execOps.exec(command, args, options);
  }

  async readFile(path: string, encoding: string = 'utf8') {
    return this.fsOps.readFile(path, encoding);
  }

  async writeFile(path: string, content: string | Buffer) {
    await this.fsOps.writeFile(path, content);
  }

  async exists(path: string) {
    return this.fsOps.exists(path);
  }

  async readdir(path: string) {
    return this.fsOps.readdir(path);
  }

  async mkdir(path: string, recursive: boolean = false) {
    await this.fsOps.mkdir(path, recursive);
  }

  async rm(path: string, recursive: boolean = false) {
    await this.fsOps.rm(path, recursive);
  }

  async stat(path: string) {
    return this.fsOps.stat(path);
  }
}
