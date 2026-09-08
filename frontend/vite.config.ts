import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  // Configure the base path for assets
  // This should match the path where the Go server will serve the static files
  base: '/',
  build: {
    // Output to a directory that the Go server will serve
    outDir: '../static',
    emptyOutDir: true,
    // Generate a manifest file for the Go server to use
    manifest: true,
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // Split Material-UI into its own chunk
          if (id.includes('@mui/material') || id.includes('@mui/icons-material') || 
              id.includes('@emotion/react') || id.includes('@emotion/styled')) {
            return 'mui';
          }
          // Split React and related libraries
          if (id.includes('react') || id.includes('react-dom') || id.includes('react-router-dom')) {
            return 'react-vendor';
          }
          // Split TanStack Query
          if (id.includes('@tanstack/react-query')) {
            return 'query';
          }
          // Split utility libraries
          if (id.includes('date-fns')) {
            return 'utils';
          }
        }
      }
    }
  },
  server: {
    // Configure the dev server to proxy API requests to the Go server
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false
      }
    }
  }
});
