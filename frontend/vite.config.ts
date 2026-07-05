import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		// Allow access via ngrok tunnels (subdomain is random per session on the free tier).
		allowedHosts: ['.ngrok-free.dev', '.ngrok-free.app', '.ngrok.io'],
		proxy: {
			'/api': 'http://localhost:30001'
		},
		// Vite normally compiles each route's module on first visit, which reads
		// as ~0.5-1s of lag per page the first time you click into it during a
		// dev session. Pre-transform everything at startup instead.
		warmup: {
			clientFiles: ['./src/routes/**/*.svelte', './src/routes/**/*.ts', './src/lib/**/*.svelte', './src/lib/**/*.ts']
		}
	},
	preview: {
		// Same proxy as dev, so `npm run preview` (a real production build) can
		// also be used to sanity-check performance without a full nginx setup.
		proxy: {
			'/api': 'http://localhost:30001'
		}
	}
});
