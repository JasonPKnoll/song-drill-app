<script lang="ts">
	import { parseFurigana } from '$lib/furigana';

	let { furi, class: className = '' }: { furi: string; class?: string } = $props();

	let segments = $derived(parseFurigana(furi));
</script>

<span class={className}>
	{#each segments as seg, i (i)}
		{#if seg.type === 'ruby'}
			<ruby>{seg.base}<rt>{seg.reading}</rt></ruby>
		{:else}
			{seg.value}
		{/if}
	{/each}
</span>
