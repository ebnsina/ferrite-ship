<script lang="ts">
	import type { Snippet } from 'svelte';
	import Button from './Button.svelte';

	interface Props {
		open: boolean;
		title: string;
		description: string;
		confirmLabel?: string;
		cancelLabel?: string;
		/** Destructive actions get the danger treatment. */
		danger?: boolean;
		busy?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
		/** Extra content, e.g. a warning about what will be lost. */
		children?: Snippet;
	}

	let {
		open = $bindable(),
		title,
		description,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		danger = false,
		busy = false,
		onConfirm,
		onCancel,
		children
	}: Props = $props();

	let dialog = $state<HTMLDialogElement | null>(null);

	// <dialog> gives focus trapping, Escape handling and inertness for free —
	// all things that are easy to reimplement badly.
	$effect(() => {
		if (!dialog) return;
		if (open && !dialog.open) dialog.showModal();
		if (!open && dialog.open) dialog.close();
	});
</script>

<dialog
	bind:this={dialog}
	onclose={onCancel}
	oncancel={onCancel}
	class="border-border bg-surface text-content m-auto w-[min(28rem,calc(100vw-2rem))] rounded-card border p-0 backdrop:bg-black/40 backdrop:backdrop-blur-sm"
>
	<div class="p-6">
		<h2 class="text-content text-base font-medium">{title}</h2>
		<p class="text-content-muted mt-2 text-sm leading-relaxed">{description}</p>

		{#if children}
			<div class="mt-4">{@render children()}</div>
		{/if}

		<div class="mt-7 flex justify-end gap-3">
			<Button variant="ghost" onclick={onCancel} disabled={busy}>{cancelLabel}</Button>
			<Button variant={danger ? 'danger' : 'primary'} onclick={onConfirm} disabled={busy}>
				{busy ? 'Working…' : confirmLabel}
			</Button>
		</div>
	</div>
</dialog>
