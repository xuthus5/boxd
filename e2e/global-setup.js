import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { request } from '@playwright/test';
import { E2E_BASE_URL, E2E_PASSWORD } from './support/env.js';

const authState = path.resolve('test-results/.auth/admin.json');

async function expectJSON(response, operation, expectedStatuses = [200, 201]) {
  if (!expectedStatuses.includes(response.status())) {
    throw new Error(`${operation} failed (${response.status()}): ${await response.text()}`);
  }
  return response.json();
}

export default async function globalSetup() {
  const context = await request.newContext({
    baseURL: E2E_BASE_URL,
    extraHTTPHeaders: { Accept: 'application/json' },
  });

  try {
    const status = await expectJSON(await context.get('/api/v1/auth/status'), 'auth status');
    const authResponse = status.initialized
      ? await context.post('/api/v1/auth/login', { data: { password: E2E_PASSWORD } })
      : await context.post('/api/v1/auth/initialize', { data: { password: E2E_PASSWORD } });
    await expectJSON(authResponse, 'authentication');

    await expectJSON(await context.patch('/api/v1/routes/presets', {
      data: { bypass_lan: false, china_direct: false, ad_block: false },
    }), 'routing preset isolation');

    await mkdir(path.dirname(authState), { recursive: true });
    await context.storageState({ path: authState });

    const existingNodes = await expectJSON(await context.get('/api/v1/nodes'), 'node list');
    if (!existingNodes.some((node) => node.name === 'E2E Tokyo')) {
      const imported = await expectJSON(await context.post('/api/v1/nodes/import-links', {
        data: {
          links: [
            'vless://9b1c4d6e-0000-0000-0000-000000000101@tokyo.e2e.invalid:443?security=tls&sni=tokyo.e2e.invalid#E2E%20Tokyo',
            'trojan://e2e-secret@singapore.e2e.invalid:443?sni=singapore.e2e.invalid#E2E%20Singapore',
          ],
        },
      }), 'node seed');
      if (imported.added !== 2) throw new Error(`expected 2 seeded nodes, got ${imported.added}`);
    }

    const nodes = await expectJSON(await context.get('/api/v1/nodes'), 'seeded node list');
    const tokyo = nodes.find((node) => node.name === 'E2E Tokyo');
    const singapore = nodes.find((node) => node.name === 'E2E Singapore');
    if (!tokyo || !singapore) throw new Error('E2E nodes were not seeded');

    const groups = await expectJSON(await context.get('/api/v1/groups'), 'group list');
    if (!groups.some((group) => group.name === 'E2E Auto')) {
      await expectJSON(await context.post('/api/v1/groups', {
        data: {
          name: 'E2E Auto',
          kind: 'selector',
          members: [tokyo.id, singapore.id],
          selected: tokyo.id,
          interval_seconds: 180,
        },
      }), 'group seed');
    }

    const chains = await expectJSON(await context.get('/api/v1/chains'), 'chain list');
    if (!chains.some((chain) => chain.name === 'E2E Relay')) {
      await expectJSON(await context.post('/api/v1/chains', {
        data: {
          name: 'E2E Relay',
          hops: [
            { kind: 'node', id: tokyo.id },
            { kind: 'node', id: singapore.id },
          ],
        },
      }), 'chain seed');
    }

    const subscriptions = await expectJSON(await context.get('/api/v1/subscriptions'), 'subscription list');
    if (!subscriptions.some((subscription) => subscription.name === 'E2E Subscription')) {
      await expectJSON(await context.post('/api/v1/subscriptions', {
        data: {
          name: 'E2E Subscription',
          url: 'https://subscription.e2e.invalid/api?token=fixture-not-real',
          interval_seconds: 21600,
          enabled: false,
        },
      }), 'subscription seed');
    }
  } finally {
    await context.dispose();
  }
}
