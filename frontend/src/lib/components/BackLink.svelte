<script lang="ts">
	let { href, label }: { href: string; label: string } = $props();
</script>

<a href={href} class="mb-4 inline-flex items-center gap-1 text-sm text-muted transition hover:text-accent">
	<span aria-hidden="true">←</span>
	{label}
</a>
