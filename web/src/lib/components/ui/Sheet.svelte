<script lang="ts">
	import { cn } from '$utils/cn';
	import X from '@lucide/svelte/icons/x';
	import type { Snippet } from 'svelte';
	import Button from './Button.svelte';

	/**
	 * A panel that slides in from the right for anything with a form in it.
	 *
	 * Three fixed regions: a header that says what this is, a content area that
	 * scrolls, and a footer that does not. Long forms are the reason — when the
	 * buttons live at the end of the document you have to scroll past every
	 * field to reach them, and you cannot see what you are about to confirm
	 * while you confirm it.
	 *
	 * Built on <dialog>, which brings focus trapping, Escape, inertness for the
	 * page behind, and top-layer stacking. Reimplementing any of that by hand
	 * is how dialogs end up unusable with a keyboard.
	 */
	interface Props {
		open: boolean;
		title: string;
		description?: string;
		/** Wider for forms with side-by-side fields. */
		size?: 'md' | 'lg';
		/** Blocks closing while something is in flight. */
		busy?: boolean;
		onClose: () => void;
		children: Snippet;
		/** Actions. Right-aligned; put the primary one last. */
		footer?: Snippet;
	}

	let {
		open = $bindable(),
		title,
		description,
		size = 'md',
		busy = false,
		onClose,
		children,
		footer
	}: Props = $props();

	let dialog = $state<HTMLDialogElement | null>(null);
	let body = $state<HTMLDivElement | null>(null);

	const WIDTHS = {
		md: 'sm:max-w-xl',
		lg: 'sm:max-w-3xl'
	} as const;

	$effect(() => {
		if (!dialog) return;
		if (open && !dialog.open) dialog.showModal();
		if (!open && dialog.open) dialog.close();
	});

	// Open at the top, always.
	//
	// Without this the sheet appears scrolled to the middle of the form: a
	// radio group whose value is set on mount gets scrolled into view by the
	// browser, so a form with a default choice halfway down opens halfway
	// down. Reset after the DOM has settled rather than before it.
	$effect(() => {
		if (!open) return;
		requestAnimationFrame(() => body?.scrollTo({ top: 0 }));
	});

	function requestClose(event?: Event) {
		// Half-filled forms and in-flight requests are worth protecting from a
		// stray Escape; the caller decides when it is safe to leave.
		if (busy) {
			event?.preventDefault();
			return;
		}
		onClose();
	}
</script>

<dialog
	bind:this={dialog}
	oncancel={requestClose}
	onclose={() => open && requestClose()}
	class={cn(
		'text-content m-0 ml-auto h-dvh max-h-dvh w-full bg-transparent p-0',
		'backdrop:bg-black/50 backdrop:backdrop-blur-sm',
		WIDTHS[size]
	)}
>
	{#if open}
		<!-- grid-rows so the middle region is the only thing that scrolls: the
		     header and the actions stay put however long the form gets. -->
		<div
			class="border-border bg-surface grid h-dvh grid-rows-[auto_1fr_auto] border-l sm:rounded-l-panel"
		>
			<header class="border-border flex items-start gap-4 border-b px-6 py-5">
				<div class="min-w-0 flex-1">
					<h2 class="text-content text-base font-medium">{title}</h2>
					{#if description}
						<p class="text-content-muted mt-1 text-sm leading-relaxed">{description}</p>
					{/if}
				</div>

				<Button
					variant="ghost"
					size="sm"
					onclick={() => requestClose()}
					disabled={busy}
					aria-label="Close"
				>
					<X size={16} aria-hidden="true" />
				</Button>
			</header>

			<div bind:this={body} class="overflow-y-auto px-6 py-6">
				{@render children()}
			</div>

			{#if footer}
				<footer class="border-border bg-surface flex flex-wrap justify-end gap-3 border-t px-6 py-4">
					{@render footer()}
				</footer>
			{/if}
		</div>
	{/if}
</dialog>

<style>
	/* The panel arrives from the edge it is anchored to. Anyone who has asked
	   not to see motion gets it without the slide. */
	dialog[open] {
		animation: slide-in 220ms cubic-bezier(0.2, 0, 0, 1);
	}

	@keyframes slide-in {
		from {
			transform: translateX(1.5rem);
			opacity: 0;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		dialog[open] {
			animation: none;
		}
	}
</style>
