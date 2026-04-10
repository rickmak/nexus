import { connectSDKClient, getDaemonEnvConfig } from '../harness/daemon';
import { startSession } from '../harness/session';
import { assertCapabilitiesArray, skipTest } from '../harness/assertions';

const hasDaemonEnv = (): boolean => getDaemonEnvConfig() !== null;

const runningInCI = (): boolean => process.env.CI === 'true';

const maybeIt = hasDaemonEnv() || runningInCI() ? it : it.skip;

describe('sdk-runtime e2e harness', () => {
  maybeIt('connects to daemon using @nexus/sdk', async () => {
    const env = getDaemonEnvConfig();
    if (env) {
      const client = await connectSDKClient(env);
      try {
        const caps = await client.workspace.capabilities();
        assertCapabilitiesArray(caps);
      } finally {
        await client.disconnect();
      }
      return;
    }

    if (!runningInCI()) {
      skipTest('daemon env not configured (NEXUS_DAEMON_WS/NEXUS_DAEMON_TOKEN); skipping harness connectivity check');
      return;
    }

    const session = await startSession({ forceManaged: true });
    try {
      const caps = await session.client.workspace.capabilities();
      assertCapabilitiesArray(caps);
    } finally {
      await session.stop();
    }
  });
});
