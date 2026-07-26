<script lang="ts">
	import { Button } from '$components/ui';
	import { toolsClient, type Preset, type SavedQuery, type Tool } from '$lib/data/tools';
	import { toAppError } from '$lib/errors';
	import BookmarkPlus from '@lucide/svelte/icons/bookmark-plus';
	import Astroid from '@lucide/svelte/icons/astroid';
	import X from '@lucide/svelte/icons/x';

	/**
	 * The queries you can start from: the ones we ship, and the ones you keep.
	 *
	 * Presets exist because the useful questions — what is in here, how big is
	 * it, what is it doing — are answered by each database's own system tables,
	 * which is exactly the knowledge someone new to that database lacks.
	 *
	 * They carry the astroid mark, which is this product's icon for anything
	 * that suggests something to you rather than doing what you asked.
	 */
	interface Props {
		serverId: string;
		tool: Tool;
		/** The editor's contents, so "Save this one" knows what to save. */
		current: string;
		onPick: (query: string) => void;
	}

	let { serverId, tool, current, onPick }: Props = $props();

	let saved = $state<SavedQuery[]>([]);
	let naming = $state(false);
	let name = $state('');
	let saveError = $state<string | null>(null);
	let busy = $state(false);

	const presets = $derived<Preset[]>(tool.consolePresets ?? []);

	async function load() {
		try {
			saved = await toolsClient.savedQueries(serverId, tool.id);
		} catch {
			// A library that will not load is not worth an error banner over the
			// console itself — the editor still works without it.
			saved = [];
		}
	}

	async function save() {
		if (busy || !name.trim() || !current.trim()) return;

		busy = true;
		saveError = null;

		try {
			await toolsClient.saveQuery(serverId, tool.id, name.trim(), current);
			await load();
			naming = false;
			name = '';
		} catch (cause) {
			saveError = toAppError(cause).message;
		} finally {
			busy = false;
		}
	}

	async function remove(entry: SavedQuery) {
		try {
			await toolsClient.deleteSavedQuery(serverId, tool.id, entry.id);
			await load();
		} catch (cause) {
			saveError = toAppError(cause).message;
		}
	}

	$effect(() => {
		void tool.id;
		void load();
	});
</script>

<div class="space-y-3">
	{#if presets.length > 0}
		<div class="flex flex-wrap items-center gap-2">
			<span class="text-content-muted flex items-center gap-1.5 text-xs font-medium">
				<Astroid size={13} aria-hidden="true" />
				Start with
			</span>

			{#each presets as preset (preset.label)}
				<button
					type="button"
					onclick={() => onPick(preset.query)}
					title={preset.description}
					class="border-border text-content-muted hover:border-border-strong hover:text-content rounded-pill border px-3 py-1 text-xs transition-colors duration-150"
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}

	<div class="flex flex-wrap items-center gap-2">
		<span class="text-content-muted flex items-center gap-1.5 text-xs font-medium">
			<BookmarkPlus size={13} aria-hidden="true" />
			Saved
		</span>

		{#each saved as entry (entry.id)}
			<span
				class="border-border rounded-pill flex items-center gap-1 border pr-1 pl-3 text-xs"
			>
				<button
					type="button"
					onclick={() => onPick(entry.query)}
					title={entry.query}
					class="text-content-muted hover:text-content py-1 transition-colors duration-150"
				>
					{entry.name}
				</button>
				<button
					type="button"
					onclick={() => remove(entry)}
					aria-label="Forget {entry.name}"
					class="text-content-subtle hover:text-error rounded-pill p-1 transition-colors duration-150"
				>
					<X size={12} aria-hidden="true" />
				</button>
			</span>
		{:else}
			<span class="text-content-subtle text-xs">Nothing yet.</span>
		{/each}

		{#if naming}
			<!-- An inline field rather than a browser prompt: a modal that steals
			     focus to ask one short question is worse than the question. -->
			<form
				class="flex items-center gap-2"
				onsubmit={(event) => {
					event.preventDefault();
					void save();
				}}
			>
				<!-- svelte-ignore a11y_autofocus -->
				<input
					bind:value={name}
					autofocus
					placeholder="Call it what?"
					aria-label="A name for this query"
					class="border-border bg-surface-sunken text-content placeholder:text-content-subtle rounded-pill h-7 border px-3 text-xs outline-none"
					onkeydown={(event) => {
						if (event.key === 'Escape') {
							naming = false;
							name = '';
						}
					}}
				/>
				<Button type="submit" size="sm" disabled={busy || !name.trim()}>Save</Button>
			</form>
		{:else}
			<Button
				variant="ghost"
				size="sm"
				onclick={() => (naming = true)}
				disabled={!current.trim()}
				title={current.trim() ? 'Keep what is in the editor' : 'Type a query first'}
			>
				Save this one
			</Button>
		{/if}
	</div>

	{#if saveError}
		<p class="text-error text-xs">{saveError}</p>
	{/if}
</div>
