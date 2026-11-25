import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';
import YAML from 'yaml';

const root = process.cwd();
const specPath = path.join(root, 'docs', 'openapi.yaml');
const handlerDir = path.join(root, 'internal', 'adapters', 'inbound', 'httpapi');
const methods = new Set(['get', 'post', 'put', 'patch', 'delete']);

const spec = YAML.parse(await readFile(specPath, 'utf8'));
if (spec.openapi !== '3.1.0') throw new Error('docs/openapi.yaml must use OpenAPI 3.1.0');
if (!spec.info?.title || !spec.info?.version) throw new Error('OpenAPI info.title and info.version are required');
if (!spec.components?.securitySchemes?.cookieAuth || !spec.components?.securitySchemes?.bearerAuth) {
  throw new Error('OpenAPI must define cookieAuth and bearerAuth security schemes');
}

const openAPIRoutes = new Set();
for (const [route, item] of Object.entries(spec.paths || {})) {
  for (const method of Object.keys(item || {})) {
    if (methods.has(method)) openAPIRoutes.add(`${method.toUpperCase()} ${route}`);
  }
}

const registeredRoutes = new Set();
const routePattern = /mux\.HandleFunc\("([A-Z]+) ([^" ]+)"/g;
for (const file of await readdir(handlerDir)) {
  if (!file.endsWith('.go') || file.endsWith('_test.go')) continue;
  const source = await readFile(path.join(handlerDir, file), 'utf8');
  for (const match of source.matchAll(routePattern)) registeredRoutes.add(`${match[1]} ${match[2]}`);
}

const missingFromSpec = [...registeredRoutes].filter((route) => !openAPIRoutes.has(route)).sort();
const staleInSpec = [...openAPIRoutes].filter((route) => !registeredRoutes.has(route)).sort();
if (missingFromSpec.length || staleInSpec.length) {
  if (missingFromSpec.length) console.error(`Missing from OpenAPI:\n${missingFromSpec.join('\n')}`);
  if (staleInSpec.length) console.error(`No matching Go route:\n${staleInSpec.join('\n')}`);
  process.exit(1);
}

const schemas = new Set(Object.keys(spec.components?.schemas || {}));
const refs = new Set([...JSON.stringify(spec).matchAll(/#\/components\/schemas\/([A-Za-z0-9_]+)/g)].map((match) => match[1]));
const missingSchemas = [...refs].filter((name) => !schemas.has(name));
if (missingSchemas.length) {
  console.error(`Missing schema definitions: ${missingSchemas.join(', ')}`);
  process.exit(1);
}

const ids = new Set();
for (const [route, item] of Object.entries(spec.paths || {})) {
  for (const [method, operation] of Object.entries(item || {})) {
    if (!methods.has(method)) continue;
    if (!operation.operationId) throw new Error(`${method.toUpperCase()} ${route} is missing operationId`);
    if (ids.has(operation.operationId)) throw new Error(`duplicate operationId: ${operation.operationId}`);
    ids.add(operation.operationId);
    if (!operation.responses || Object.keys(operation.responses).length === 0) {
      throw new Error(`${method.toUpperCase()} ${route} has no responses`);
    }
  }
}

console.log(`OpenAPI contract verified: ${registeredRoutes.size} routes, ${schemas.size} schemas.`);
