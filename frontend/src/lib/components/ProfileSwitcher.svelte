<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listProfiles,
		getActiveProfile,
		setActiveProfile,
		createProfile,
		renameProfile,
		deleteProfile,
		type Profile
	} from '$lib/api';
	import { cn } from '$lib/utils/cn';

	const SWATCHES = ['#a78bfa', '#6ee7a0', '#f87171', '#fbbf24', '#7dd3fc'];

	let profiles = $state<Profile[]>([]);
	let active = $state<Profile | null>(null);
	let open = $state(false);
	let mode = $state<'list' | 'create' | 'edit'>('list');
	let editingId = $state<number | null>(null);
	let formName = $state('');
	let formColor = $state(SWATCHES[0]);
	let error = $state<string | null>(null);
	let rootEl: HTMLDivElement;

	onMount(() => {
		refresh();

		function onClickOutside(e: MouseEvent) {
			// composedPath (not e.target/contains) — a click that itself causes
			// a re-render (e.g. "+ New profile" swapping the list for a form)
			// can detach the original target from the DOM before this listener
			// runs, making contains() wrongly report "outside" for a click that
			// was very much inside. composedPath is captured at dispatch time,
			// so it stays accurate regardless of later mutations.
			if (open && rootEl && !e.composedPath().includes(rootEl)) {
				open = false;
				mode = 'list';
			}
		}
		document.addEventListener('click', onClickOutside);
		return () => document.removeEventListener('click', onClickOutside);
	});

	async function refresh() {
		const [list, current] = await Promise.all([listProfiles(), getActiveProfile()]);
		profiles = list;
		active = current;
	}

	function toggleOpen() {
		open = !open;
		mode = 'list';
		error = null;
	}

	async function switchTo(id: number) {
		if (active?.id === id) {
			open = false;
			return;
		}
		await setActiveProfile(id);
		// Every page's drill queue / stats / mastered counts are scoped to the
		// active profile — a full reload is the simplest way to make sure
		// nothing on screen is left showing the previous profile's data.
		window.location.reload();
	}

	function startCreate() {
		mode = 'create';
		formName = '';
		formColor = SWATCHES[Math.floor(Math.random() * SWATCHES.length)];
		error = null;
	}

	function startEdit(p: Profile) {
		mode = 'edit';
		editingId = p.id;
		formName = p.display_name;
		formColor = p.color;
		error = null;
	}

	async function save() {
		if (!formName.trim()) {
			error = 'Name is required.';
			return;
		}
		const wasEditingActive = mode === 'edit' && active?.id === editingId;
		try {
			if (mode === 'create') {
				await createProfile(formName.trim(), formColor);
			} else if (mode === 'edit' && editingId !== null) {
				await renameProfile(editingId, formName.trim(), formColor);
			}
			if (wasEditingActive) {
				// The switcher button shows the active profile's name/color —
				// simplest to reload rather than juggle in-place state sync.
				window.location.reload();
				return;
			}
			await refresh();
			mode = 'list';
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	function initials(name: string): string {
		return name.slice(0, 2).toUpperCase();
	}

	async function remove(p: Profile) {
		if (!confirm(`Delete profile "${p.display_name}"? This removes all of its progress.`)) return;
		try {
			await deleteProfile(p.id);
			const wasActive = active?.id === p.id;
			await refresh();
			if (wasActive) {
				window.location.reload();
			}
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}
</script>

<div class="relative" bind:this={rootEl}>
	<button
		type="button"
		class={cn(
			'flex h-9 w-9 items-center justify-center',
			'text-bg text-sm font-bold tracking-wide',
			'rounded-full',
			'transition hover:opacity-90'
		)}
		style="background-color: {active?.color ?? '#a78bfa'}B3"
		onclick={toggleOpen}
		aria-expanded={open}
		aria-label="Switch profile{active ? ` (currently ${active.display_name})` : ''}"
	>
		{active ? initials(active.display_name) : ''}
	</button>

	{#if open}
		<div
			class={cn(
				'absolute top-full right-0 z-10 mt-2 w-64',
				'border border-border bg-surface',
				'rounded-2xl shadow-lg',
				'p-2'
			)}
		>
			{#if mode === 'list'}
				<ul class="flex flex-col gap-1">
					{#each profiles as p (p.id)}
						<li class="flex items-center gap-2 rounded-xl px-2 py-1.5 hover:bg-border/40">
							<button
								type="button"
								class="flex flex-1 items-center gap-2 text-left"
								onclick={() => switchTo(p.id)}
							>
								<span class="h-4 w-4 shrink-0 rounded-full" style="background-color: {p.color}"
								></span>
								<span class={cn('text-sm', p.id === active?.id ? 'text-ink font-medium' : 'text-muted')}>
									{p.display_name}
								</span>
								{#if p.id === active?.id}
									<span class="ml-auto text-xs text-accent">active</span>
								{/if}
							</button>
							<button
								type="button"
								class="text-xs text-muted hover:text-ink"
								onclick={() => startEdit(p)}
								aria-label="Edit {p.display_name}"
							>
								Edit
							</button>
							<button
								type="button"
								class="text-xs text-muted hover:text-bad"
								onclick={() => remove(p)}
								aria-label="Delete {p.display_name}"
							>
								✕
							</button>
						</li>
					{/each}
				</ul>
				<button
					type="button"
					class={cn(
						'mt-1 w-full py-1.5',
						'text-sm text-accent',
						'rounded-xl hover:bg-accent/10',
						'transition'
					)}
					onclick={startCreate}
				>
					+ New profile
				</button>
			{:else}
				<div class="flex flex-col gap-2 p-1">
					<input
						type="text"
						placeholder="Name"
						class={cn(
							'px-3 py-2',
							'border border-border bg-bg text-ink',
							'rounded-xl text-sm',
							'focus:outline-none focus:border-accent'
						)}
						bind:value={formName}
					/>
					<div class="flex gap-2">
						{#each SWATCHES as swatch}
							<button
								type="button"
								class={cn(
									'h-6 w-6 rounded-full',
									'transition',
									formColor === swatch ? 'ring-2 ring-ink ring-offset-2 ring-offset-surface' : ''
								)}
								style="background-color: {swatch}"
								onclick={() => (formColor = swatch)}
								aria-label="Color {swatch}"
							></button>
						{/each}
					</div>
					{#if error}
						<p class="text-xs text-bad">{error}</p>
					{/if}
					<div class="mt-1 flex gap-2">
						<button
							type="button"
							class={cn(
								'flex-1 py-1.5',
								'text-sm text-muted',
								'border border-border rounded-xl',
								'hover:bg-border/40 transition'
							)}
							onclick={() => (mode = 'list')}
						>
							Cancel
						</button>
						<button
							type="button"
							class={cn(
								'flex-1 py-1.5',
								'text-sm text-bg font-medium',
								'bg-accent rounded-xl',
								'hover:opacity-90 transition'
							)}
							onclick={save}
						>
							Save
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>
