import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		// Allow access via ngrok tunnels (subdomain is random per session on the free tier).
		allowedHosts: ['.ngrok-free.dev', '.ngrok-free.app', '.ngrok.io']
	}
});
