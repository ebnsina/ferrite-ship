<script lang="ts">
	import { Button, Card } from '$components/ui';
	import { filesClient, type FileContent } from '$lib/data/files';
	import { toAppError } from '$lib/errors';
	import { formatBytes } from '$utils/format';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Download from '@lucide/svelte/icons/download';
	import Save from '@lucide/svelte/icons/save';

	interface Props {
		serverId: string;
		file: FileContent;
		onClose: () => void;
	}

	let { serverId, file, onClose }: Props = $props();

	let draft = $state('');
	// What is on the server, so "unsaved" means something after a save without
	// mutating the prop we were handed.
	let baseline = $state('');
	let saving = $state(false);
	let error = $state<string | null>(null);
	let savedAt = $state<number | null>(null);

	// Re-seeds whenever a different file is opened, rather than capturing only
	// whichever file happened to be first.
	$effect(() => {
		const incoming = file.text;
		draft = incoming;
		baseline = incoming;
		savedAt = null;
		error = null;
	});

	const dirty = $derived(draft !== baseline);

	async function save() {
		if (saving || !dirty) return;
		saving = true;
		error = null;

		try {
			await filesClient.write(serverId, file.path, draft);
			baseline = draft;
			savedAt = Date.now();
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			saving = false;
		}
	}

	// Cmd/Ctrl+S is what everyone's fingers already do in an editor.
	function onKeydown(event: KeyboardEvent) {
		if ((event.metaKey || event.ctrlKey) && event.key === 's') {
			event.preventDefault();
			void save();
		}
	}
</script>

<svelte:window onkeydown={onKeydown} />

<Card padded={false} class="overflow-hidden">
	<div class="border-border/70 flex flex-wrap items-center gap-3 border-b px-5 py-3">
		<Button variant="ghost" size="sm" onclick={onClose}>
			<ArrowLeft size={15} aria-hidden="true" />
			Back
		</Button>

		<div class="min-w-0 flex-1">
			<p class="text-content font-machine truncate text-xs">{file.path}</p>
			<p class="text-content-subtle mt-0.5 text-xs">
				{formatBytes(file.size)} · {file.mode}
				{#if dirty}
					· <span class="text-warn">unsaved changes</span>
				{:else if savedAt}
					· <span class="text-ok">saved</span>
				{/if}
			</p>
		</div>

		<a
			href={filesClient.downloadUrl(serverId, file.path)}
			class="text-content-muted hover:bg-surface-raised hover:text-content inline-flex size-8 items-center justify-center rounded-tile transition-colors duration-150"
			aria-label="Download this file"
		>
			<Download size={15} aria-hidden="true" />
		</a>

		<Button size="sm" onclick={save} disabled={!dirty || saving}>
			<Save size={15} aria-hidden="true" />
			{saving ? 'Saving…' : 'Save'}
		</Button>
	</div>

	{#if error}
		<p class="bg-error-soft text-error border-border/70 border-b px-5 py-3 text-sm">{error}</p>
	{/if}

	<label class="sr-only" for="file-contents">Contents of {file.path}</label>
	<textarea
		id="file-contents"
		bind:value={draft}
		spellcheck={false}
		autocomplete="off"
		autocapitalize="off"
		class="font-machine bg-surface-sunken text-content h-[60vh] min-h-80 w-full resize-none px-5 py-4 text-xs leading-relaxed outline-none"
	></textarea>
</Card>
