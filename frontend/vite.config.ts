import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  test: {
    exclude: ['e2e/**', 'node_modules/**', 'dist/**'],
  },
  build: {
    outDir: '../web',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return undefined
          }
          if (id.includes('/vuetify/')) {
            return 'vendor-vuetify'
          }
          if (id.includes('/keycloak-js/')) {
            return 'vendor-keycloak'
          }
          if (id.includes('/@connectrpc/') || id.includes('/@bufbuild/')) {
            return 'vendor-connect'
          }
          if (id.includes('/vue/') || id.includes('/vue-router/') || id.includes('/pinia/')) {
            return 'vendor-vue'
          }
          return 'vendor'
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:18080',
    },
  },
})
