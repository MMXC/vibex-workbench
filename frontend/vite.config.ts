import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

/**
 * HMR / 端口说明：
 * - 请只用「一个」地址访问 dev（默认 http://localhost:5173）。
 * - 若浏览器在 :5174、Vite 在 :5173（代理/端口转发），WebSocket 可能对不上 → 400。
 *   做法：直连 Vite 端口，或给反向代理开启 WS 转发，或设置 hmr.clientPort / hmr.host
 *   见 https://vite.dev/config/server-options.html#server-hmr
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
			},
		},
	},
});
