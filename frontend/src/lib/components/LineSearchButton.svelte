<script lang="ts">
	import { blur } from 'svelte/transition';
	import Search from '@lucide/svelte/icons/search';
	import { cn } from '$lib/utils/cn';

	let { href }: { href: string } = $props();
</script>

<a
	{href}
	class={cn(
		'absolute top-3 right-3 flex h-10 w-10 items-center justify-center',
		'bg-accent text-bg shadow-black/30',
		'rounded-full shadow-lg',
		'transition-transform hover:scale-105 active:scale-95'
	)}
	onclick={(e) => e.stopPropagation()}
	aria-label="Look up this line's vocab"
	in:blur={{ duration: 260, amount: 7 }}
	out:blur={{ duration: 220, amount: 7 }}
>
	<Search size={18} strokeWidth={2.5} />
</a>
