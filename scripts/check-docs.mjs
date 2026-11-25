import { existsSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();
const requiredFiles = [
  'README.md',
  'CHANGELOG.md',
  'SECURITY.md',
  'CONTRIBUTING.md',
  'CODE_OF_CONDUCT.md',
  'LICENSE',
  'NOTICE',
  'docs/compatibility.md',
  'docs/control-plane.md',
  'docs/openapi.yaml',
  'docs/release-checklist.md',
  'docs/sing-box-versioning.md',
  'docs/images/overview.png',
  'docs/images/egress-chains.png',
  'docs/images/subscriptions.png',
  'docs/images/logs.png',
  'docs/images/overview-mobile.png',
];

const documentation = [
  'README.md',
  'CHANGELOG.md',
  'SECURITY.md',
  'CONTRIBUTING.md',
  'CODE_OF_CONDUCT.md',
  'docs/compatibility.md',
  'docs/control-plane.md',
  'docs/release-checklist.md',
  'docs/sing-box-versioning.md',
];

for (const file of requiredFiles) {
  if (!existsSync(path.join(root, file))) throw new Error(`required release document is missing: ${file}`);
}

let checkedLinks = 0;
for (const file of documentation) {
  const source = await readFile(path.join(root, file), 'utf8');
  for (const match of source.matchAll(/\]\(([^)#]+)(?:#[^)]*)?\)/g)) {
    const target = match[1];
    if (/^(?:https?:|mailto:)/.test(target)) continue;
    checkedLinks += 1;
    if (!existsSync(path.resolve(root, path.dirname(file), target))) {
      throw new Error(`${file} links to a missing local target: ${target}`);
    }
  }
}

console.log(`Release documentation verified: ${requiredFiles.length} required files, ${checkedLinks} local links.`);
