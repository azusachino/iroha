#!/usr/bin/env node
import { mkdir, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const openAPIURL = process.env.POSTGREST_OPENAPI_URL ?? 'http://127.0.0.1:3001/';
const output = resolve('src/lib/postgrest/types.ts');
const tmpBase = join(tmpdir(), 'iroha-postgrest-openapi');
const swaggerPath = `${tmpBase}-swagger.json`;
const openAPIPath = `${tmpBase}-openapi.json`;

async function main() {
	const response = await fetch(openAPIURL, { headers: { accept: 'application/openapi+json' } });
	if (!response.ok) {
		throw new Error(`failed to fetch ${openAPIURL}: ${response.status} ${response.statusText}`);
	}

	await mkdir(dirname(output), { recursive: true });
	await writeFile(swaggerPath, await response.text());

	run('swagger2openapi', [swaggerPath, '-o', openAPIPath]);
	run('openapi-typescript', [openAPIPath, '-o', output]);
}

function run(command, args) {
	const result = spawnSync(command, args, { stdio: 'inherit' });
	if (result.status !== 0) {
		throw new Error(`${command} exited with status ${result.status}`);
	}
}

main().catch((error) => {
	console.error(error instanceof Error ? error.message : error);
	process.exit(1);
});
