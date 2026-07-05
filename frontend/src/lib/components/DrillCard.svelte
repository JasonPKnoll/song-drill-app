<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		front,
		back,
		onGotIt,
		onMissed
	}: {
		front: Snippet;
		back: Snippet;
		onGotIt: () => void;
		onMissed: () => void;
	} = $props();

	let revealed = $state(false);
</script>

<div class="rounded-2xl border border-border bg-surface p-8 shadow-lg">
	<button
		type="button"
		class="flex min-h-48 w-full flex-col items-center justify-center gap-4 text-center disabled:cursor-default"
		onclick={() => (revealed = true)}
		disabled={revealed}
	>
		{@render front()}
	</button>

	{#if revealed}
		<div class="mt-6 border-t border-border pt-6">
			{@render back()}
		</div>

		<div class="mt-6 flex gap-3">
			<button
				type="button"
				class="flex-1 rounded-xl border border-bad bg-bad/10 py-3 font-medium text-bad transition hover:bg-bad/20"
				onclick={onMissed}
			>
				Missed
			</button>
			<button
				type="button"
				class="flex-1 rounded-xl border border-good bg-good/10 py-3 font-medium text-good transition hover:bg-good/20"
				onclick={onGotIt}
			>
				Got it
			</button>
		</div>
	{:else}
		<p class="mt-4 text-center text-sm text-muted">Tap to reveal</p>
	{/if}
</div>
