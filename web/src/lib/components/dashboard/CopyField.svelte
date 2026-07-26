<script lang="ts">
	import { Button } from '$components/ui';
	import Check from '@lucide/svelte/icons/check';
	import Copy from '@lucide/svelte/icons/copy';
	import Eye from '@lucide/svelte/icons/eye';
	import EyeOff from '@lucide/svelte/icons/eye-off';

	interface Props {
		label: string;
		value: string;
		/** Hides the value behind dots until asked for. Use for credentials. */
		secret?: boolean;
		/** Shown under the field, e.g. what to paste it into. */
		hint?: string;
	}

	let { label, value, secret = false, hint }: Props = $props();

	let revealed = $state(false);
	let copied = $state(false);
	let failed = $state(false);

	const shown = $derived(secret && !revealed ? '•'.repeat(Math.min(value.length, 32)) : value);

	async function copy() {
		failed = false;

		try {
			await navigator.clipboard.writeText(value);
			copied = true;
			// Long enough to notice, short enough not to look stuck.
			setTimeout(() => (copied = false), 1600);
		} catch {
			// The clipboard is refused outside a secure context, and over plain
			// HTTP on an IP address that is most of the time. Saying so beats a
			// button that silently does nothing.
			failed = true;
		}
	}
</script>

<div>
	<div class="flex items-center justify-between gap-3">
		<span class="text-content-muted text-xs font-medium">{label}</span>

		<div class="flex items-center gap-1">
			{#if secret}
				<Button
					variant="ghost"
					size="sm"
					onclick={() => (revealed = !revealed)}
					aria-label={revealed ? `Hide ${label}` : `Show ${label}`}
				>
					{#if revealed}
						<EyeOff size={14} aria-hidden="true" />
					{:else}
						<Eye size={14} aria-hidden="true" />
					{/if}
				</Button>
			{/if}

			<Button variant="ghost" size="sm" onclick={copy} aria-label="Copy {label}">
				{#if copied}
					<Check size={14} aria-hidden="true" class="text-ok" />
					Copied
				{:else}
					<Copy size={14} aria-hidden="true" />
					Copy
				{/if}
			</Button>
		</div>
	</div>

	<p
		class="border-border bg-surface-sunken text-content font-machine rounded-field mt-1.5 overflow-x-auto border px-3 py-2 text-xs whitespace-pre"
	>{shown}</p>

	{#if failed}
		<p class="text-warn mt-1.5 text-xs">
			Your browser would not let us copy that. Select it and copy it yourself.
		</p>
	{:else if hint}
		<p class="text-content-subtle mt-1.5 text-xs">{hint}</p>
	{/if}
</div>
