import * as esbuild from 'esbuild';
import { copyFileSync, mkdirSync } from 'fs';

const isWatch = process.argv.includes('--watch');
const isServe = process.argv.includes('--serve');

mkdirSync('dist', { recursive: true });

// Copy index.html to dist
copyFileSync('src/index.html', 'dist/index.html');

const opts = {
  entryPoints: ['src/index.ts'],
  bundle: true,
  outfile: 'dist/bundle.js',
  format: 'esm',
  target: ['es2020'],
  sourcemap: isWatch,
  minify: !isWatch,
  logLevel: 'info',
};

if (isWatch) {
  const ctx = await esbuild.context(opts);
  await ctx.watch();
  if (isServe) {
    const { host, port } = await ctx.serve({ servedir: 'dist' });
    console.log(`Dev server: http://${host}:${port}`);
  }
} else {
  await esbuild.build(opts);
}
