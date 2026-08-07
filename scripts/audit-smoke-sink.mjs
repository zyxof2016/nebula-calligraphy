#!/usr/bin/env node

import { appendFile } from 'node:fs/promises';
import { createServer } from 'node:http';

const port = Number(process.env.AUDIT_SMOKE_PORT || 18091);
const token = process.env.AUDIT_SMOKE_TOKEN || '';
const output = process.env.AUDIT_SMOKE_OUTPUT || '';

if (!Number.isInteger(port) || port < 1024 || port > 65535) {
  throw new Error('AUDIT_SMOKE_PORT must be an unprivileged TCP port');
}
if (!token || !output) {
  throw new Error('AUDIT_SMOKE_TOKEN and AUDIT_SMOKE_OUTPUT are required');
}

function authorized(request) {
  return request.headers.authorization === `Bearer ${token}`;
}

async function readJSON(request) {
  const chunks = [];
  let size = 0;
  for await (const chunk of request) {
    size += chunk.length;
    if (size > 64 * 1024) throw new Error('request body is too large');
    chunks.push(chunk);
  }
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}

const server = createServer(async (request, response) => {
  if (!authorized(request)) {
    response.writeHead(401).end();
    return;
  }
  if (request.method === 'GET' && request.url === '/health/ready') {
    response.writeHead(204).end();
    return;
  }
  if (request.method !== 'POST' || request.url !== '/api/v1/events') {
    response.writeHead(404).end();
    return;
  }
  if (!request.headers['content-type']?.startsWith('application/json')) {
    response.writeHead(415).end();
    return;
  }
  try {
    const event = await readJSON(request);
    if (!event.action || !event.outcome || !event.created_at) {
      response.writeHead(422).end();
      return;
    }
    await appendFile(output, `${JSON.stringify(event)}\n`, { mode: 0o600 });
    response.writeHead(202).end();
  } catch {
    response.writeHead(400).end();
  }
});

server.listen(port, '127.0.0.1', () => {
  process.stdout.write(`audit smoke sink listening on 127.0.0.1:${port}\n`);
});

function shutdown() {
  server.close(() => process.exit(0));
}
process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
