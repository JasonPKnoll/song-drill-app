import http from 'node:http';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// The proxy's default agent pools (keep-alive-reuses) connections to the
// Go backend. If a pooled connection ever goes stale — including simply as
// a race against the backend's own IdleTimeout closing it server-side —
// the next request that happens to get handed that exact dead socket just
// hangs with no error and no backend-side trace, since it never actually
// gets a working connection to travel over. That's indistinguishable from
// "the app is frozen" and shows up after however many requests it takes to
// cycle onto the bad socket. A non-keep-alive agent opens a fresh TCP
// connection per proxied request instead — on localhost that costs nothing
// worth noticing, and it removes this failure mode entirely rather than
// just bounding how long it takes to surface.
const backendProxyAgent = new http.Agent({ keepAlive: false });

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		// Allow access via ngrok tunnels (subdomain is random per session on the free tier).
		allowedHosts: ['.ngrok-free.dev', '.ngrok-free.app', '.ngrok.io'],
		proxy: {
			'/api': { target: 'http://localhost:30001', agent: backendProxyAgent }
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
			'/api': { target: 'http://localhost:30001', agent: backendProxyAgent }
		}
	}
});
