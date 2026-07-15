<script lang="ts">
	import { cn } from '$lib/utils/cn';

	let { href, label }: { href: string; label: string } = $props();
</script>

<a
	href={href}
	class={cn(
		'mb-4 inline-flex items-center gap-1',
		'text-sm',
		'text-muted',
		'transition hover:text-accent',
		'no-focus-ring'
	)}
>
	<span aria-hidden="true">←</span>
	{label}
</a>
