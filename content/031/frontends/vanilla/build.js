import * as esbuild from 'esbuild';
import { mkdirSync, copyFileSync, existsSync, readdirSync, statSync } from 'node:fs';
import { join, dirname } from 'node:path';

const outDir = 'dist';
mkdirSync(outDir, { recursive: true });

const envDefine = {
  'import.meta.env.VITE_OTLP_ENDPOINT': JSON.stringify(
    process.env.VITE_OTLP_ENDPOINT || 'http://localhost:4318'
  ),
  'import.meta.env.VITE_FARO_URL': JSON.stringify(
    process.env.VITE_FARO_URL || 'http://localhost:12347/collect'
  ),
  'import.meta.env.VITE_REPLAY_ENDPOINT': JSON.stringify(
    process.env.VITE_REPLAY_ENDPOINT || 'http://localhost:8080/replay/ingest'
  ),
};

await esbuild.build({
  entryPoints: ['src/main.js'],
  bundle: true,
  format: 'esm',
  target: 'es2020',
  outfile: join(outDir, 'bundle.js'),
  sourcemap: true,
  minify: true,
  define: envDefine,
  logLevel: 'info',
});

copyFileSync('src/index.html', join(outDir, 'index.html'));
copyFileSync('src/styles.css', join(outDir, 'styles.css'));

console.log('[build] done ->', outDir);
