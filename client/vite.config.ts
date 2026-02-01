import { defineConfig } from 'vite';
import solid from 'vite-plugin-solid';

export default defineConfig({
  plugins: [solid()],
  server: {
    port: 3001,
    strictPort: true,
    host: true,
    hmr: {
      host: 'localhost',
      port: 24678,
      clientPort: 24678
    }
  },
  optimizeDeps: {
    include: ['@dagrejs/dagre', '@dagrejs/graphlib']
  }
});
