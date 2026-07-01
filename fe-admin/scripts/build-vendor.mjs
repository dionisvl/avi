import { cp, mkdir, rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.resolve(__dirname, '..');
const publicDir = path.join(rootDir, 'public');
const vendorDir = path.join(publicDir, 'vendor');

const copies = [
    ['node_modules/alpinejs/dist/cdn.min.js', 'vendor/alpinejs/cdn.min.js'],
    ['node_modules/filepond/dist/filepond.min.js', 'vendor/filepond/filepond.min.js'],
    ['node_modules/filepond/dist/filepond.min.css', 'vendor/filepond/filepond.min.css'],
    ['node_modules/tabulator-tables/dist/js/tabulator.min.js', 'vendor/tabulator/tabulator.min.js'],
    ['node_modules/tabulator-tables/dist/css/tabulator_simple.min.css', 'vendor/tabulator/tabulator_simple.min.css'],
];

await rm(vendorDir, { recursive: true, force: true });
await mkdir(path.join(publicDir, 'assets'), { recursive: true });

for (const [from, to] of copies) {
    const source = path.join(rootDir, from);
    const target = path.join(publicDir, to);

    await mkdir(path.dirname(target), { recursive: true });
    await cp(source, target);
}
