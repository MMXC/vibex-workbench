import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

/**
 * HMR / 端口说明：
 * - 请只用「一个」地址访问 dev（默认 http://localhost:5173）。
 * - 若通过端口转发/代理访问，Vite 会自动处理 HMR WebSocket，无需手动配置。
 */
const SSE_PORT = process.env.VITE_SSE_PORT || '33338';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5173,
		strictPort: false,
		proxy: {
			'/api': {
				target: `http://localhost:${SSE_PORT}`,
				changeOrigin: true,
				bypass(req) {
					// SvelteKit server-side API routes — do NOT proxy, let SvelteKit handle
					const p = req.url ?? '';
					if (p.startsWith('/api/workspace/specs/list') ||
						p.startsWith('/api/workspace/specs/convention')) {
						return p; // bypass → serve via SvelteKit
					}
					return undefined; // proxy to Go backend
				},
			},
		},
	},
});
