import { defineConfig } from 'vite';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import tailwindcss from '@tailwindcss/vite';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export default defineConfig({
    root: path.resolve(__dirname, 'public'),
    publicDir: false,
    plugins: [
        tailwindcss(),
        {
            name: 'copy-env-js',
            writeBundle() {
                const source = path.resolve(__dirname, 'env.js.template');
                const target = path.resolve(__dirname, 'dist/env.js');
                const template = fs.readFileSync(source, 'utf8');
                const apiUrl = process.env.API_URL || 'http://api.avi.test';
                fs.writeFileSync(target, template.replaceAll('${API_URL}', apiUrl));
            },
        },
    ],
    build: {
        outDir: path.resolve(__dirname, 'dist'),
        emptyOutDir: true,
        assetsDir: 'assets',
        rollupOptions: {
            input: path.resolve(__dirname, 'public/index.html'),
        },
    },
});
