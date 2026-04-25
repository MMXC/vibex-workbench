import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

/**
 * HMR / 端口说明：
 * - 请只用「一个」地址访问 dev（默认 http://localhost:5173）。
 * - 若通过端口转发/代理访问，Vite 会自动处理 HMR WebSocket，无需手动配置。
 */
const SSE_PORT = process.env.VITE_SSE_PORT || '33338';
// Wails mode: Go backend subprocess runs on this port
const BACKEND_PORT = process.env.VITE_BACKEND_PORT || '33338';

export default defineConfig({
	plugins: [sveltekit()],
	// 确保 HMR 模块缓存每次 dev 启动都干净
	cacheDir: './node_modules/.vite-dev',
	server: {
		port: 5173,
		strictPort: false,
		proxy: {
			'/api': {
				target: `http://localhost:${BACKEND_PORT}`,
				changeOrigin: true,
			},
		},
	},
});
