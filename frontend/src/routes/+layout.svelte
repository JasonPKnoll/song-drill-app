<script lang="ts">
	import '../app.css';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="min-h-screen bg-bg">
	<header class="border-b border-border">
		<div class="mx-auto max-w-3xl px-4 py-4">
			<a href="/" class="text-lg font-semibold tracking-tight text-ink">
				song<span class="text-accent">-drill</span>
			</a>
		</div>
	</header>

	<main class="mx-auto max-w-3xl px-4 py-6">
		{@render children()}
	</main>
</div>
