import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { sveltekit } from '@sveltejs/kit/vite';
import { compression } from 'vite-plugin-compression2';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    sveltekit({
      preprocess: [vitePreprocess()],
      adapter: adapter({
        pages: 'build/assets',
        assets: 'build/assets',
        fallback: 'index.html',
        precompress: false
      })
    }),
    compression({ algorithms: ['gzip', 'brotliCompress'] })
  ],
  server: {
    host: '127.0.0.1',
    port: Number(process.env.VITE_PORT || '5173'),
    strictPort: true,
    proxy: {
      '/api': {
        target: process.env.BACKEND_URL || `http://127.0.0.1:${process.env.API_PORT || '4000'}`,
        changeOrigin: true
      }
    }
  }
});
