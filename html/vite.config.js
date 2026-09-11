import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src')
      }
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: 'modern-compiler' // 使用新的 Sass API
        },
        sass: {
          api: 'modern-compiler' // 使用新的 Sass API
        }
      }
    },
    server: {
      port: 3007,
      // 启用 History API fallback，支持直接访问路由（如 /users）
      // 这对于 Vue Router 的 History 模式是必需的
      strictPort: false,
      // API 代理已禁用，使用 VITE_API_BASE_URL 直接连接
      // 但 WebSocket 需要代理，因为浏览器无法直接跨域连接 WebSocket
      // 公开图片走同源相对路径（通知 Logo 等），避免开发时 localhost:3007 -> :3000 跨域
      proxy: {
        '/api/admin/public': {
          target: env.VITE_API_BASE_URL || 'http://127.0.0.1:3000',
          changeOrigin: true,
          secure: false
        },
        '/ws': {
          target: env.VITE_API_BASE_URL || 'http://127.0.0.1:3000',
          changeOrigin: true,
          ws: true, // 启用 WebSocket 代理
          secure: false, // 如果是 https，设置为 false 允许自签名证书
          rewrite: (path) => path // 保持路径不变
        }
      }
    },
    build: {
      outDir: './dist',
      emptyOutDir: true
    },
    test: {
      environment: 'happy-dom',
      include: ['src/**/*.{test,spec}.{js,ts}']
    }
  }
})

