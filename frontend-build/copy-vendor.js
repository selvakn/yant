import { cpSync, mkdirSync } from 'fs';
import { dirname, resolve } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const vendorDir = resolve(__dirname, '../frontend/static/vendor');

mkdirSync(vendorDir, { recursive: true });

const files = [
  ['node_modules/htmx.org/dist/htmx.min.js',      'htmx.min.js'],
  ['node_modules/easymde/dist/easymde.min.js',      'easymde.min.js'],
  ['node_modules/easymde/dist/easymde.min.css',     'easymde.min.css'],
  ['node_modules/mermaid/dist/mermaid.min.js',      'mermaid.min.js'],
];

for (const [src, dest] of files) {
  const from = resolve(__dirname, src);
  const to = resolve(vendorDir, dest);
  cpSync(from, to);
  console.log(`  ${dest}`);
}

console.log(`Vendor assets copied to ${vendorDir}`);
