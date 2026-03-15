import { sveltekit } from '@sveltejs/kit/vite';
import { compression } from 'vite-plugin-compression2';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit(), compression({ algorithms: ['gzip', 'brotliCompress'] })],
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${process.env.API_PORT || '4000'}`,
        changeOrigin: true
      }
    }
  }
});
