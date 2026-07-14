<script lang="ts">
	import { cn } from '$lib/utils/cn';

	// A styled stand-in for window.confirm() — the native browser dialog
	// looks jarring against the app's own dark theme and can't be restyled.
	// Controlled: `open` decides visibility, the parent owns what happens
	// on confirm/cancel (including closing itself, usually by clearing
	// whatever "pending target" state made this dialog open in the first
	// place).
	let {
		open,
		title,
		message,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		danger = false,
		onConfirm,
		onCancel
	}: {
		open: boolean;
		title: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		danger?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	} = $props();

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onCancel();
	}
</script>

<svelte:window onkeydown={open ? onKeydown : undefined} />

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-bg/70 p-4"
		onclick={(e) => {
			if (e.target === e.currentTarget) onCancel();
		}}
		role="presentation"
	>
		<div
			class={cn(
				'w-full max-w-sm p-6',
				'border border-border bg-surface',
				'rounded-2xl shadow-lg'
			)}
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="confirm-dialog-title"
		>
			<h2 id="confirm-dialog-title" class="text-lg font-semibold text-ink">{title}</h2>
			<p class="mt-2 text-sm text-muted">{message}</p>
			<div class="mt-6 flex gap-3">
				<button
					type="button"
					class={cn(
						'flex-1 py-2.5',
						'text-sm font-medium text-muted',
						'border border-border rounded-xl',
						'transition hover:bg-border/40'
					)}
					onclick={onCancel}
				>
					{cancelLabel}
				</button>
				<button
					type="button"
					class={cn(
						'flex-1 py-2.5',
						'text-sm font-medium',
						'rounded-xl border transition',
						danger
							? 'border-bad bg-bad/10 text-bad hover:bg-bad/20'
							: 'border-accent bg-accent/10 text-accent hover:bg-accent/20'
					)}
					onclick={onConfirm}
				>
					{confirmLabel}
				</button>
			</div>
		</div>
	</div>
{/if}
