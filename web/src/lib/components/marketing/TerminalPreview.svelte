<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Minus from '@lucide/svelte/icons/minus';
	import Sparkles from '@lucide/svelte/icons/sparkles';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';

	type LineKind = 'command' | 'done' | 'already' | 'summary';

	interface Line {
		kind: LineKind;
		text: string;
	}

	// What setting up a brand new server actually looks like, in plain words.
	const lines: Line[] = [
		{ kind: 'command', text: 'Setting up edge-fra-1' },
		{ kind: 'done', text: 'Updates installed' },
		{ kind: 'done', text: 'Your login account created' },
		{ kind: 'already', text: 'Root login already turned off' },
		{ kind: 'done', text: 'Firewall on — only web traffic allowed in' },
		{ kind: 'already', text: 'Repeat-login blocking already running' },
		{ kind: 'done', text: 'Security updates set to install themselves' },
		{ kind: 'summary', text: '7 checks · 4 changed · 3 already fine · 6.2 seconds' }
	];

	const TONE: Record<LineKind, string> = {
		command: 'text-content',
		done: 'text-ok',
		already: 'text-content-subtle',
		summary: 'text-content-muted'
	};

	const ICON: Record<LineKind, Component<LucideProps>> = {
		command: ChevronRight,
		done: Check,
		already: Minus,
		summary: Sparkles
	};
</script>

<div class="border-border bg-surface-sunken overflow-hidden rounded-panel border">
	<div class="border-border/70 flex items-center gap-2 border-b px-5 py-3.5">
		<span class="bg-error/70 size-2.5 rounded-pill"></span>
		<span class="bg-warn/70 size-2.5 rounded-pill"></span>
		<span class="bg-ok/70 size-2.5 rounded-pill"></span>
		<span class="text-content-subtle ml-2 text-xs">First-time setup</span>
	</div>

	<div class="space-y-2 overflow-x-auto p-5 text-sm">
		{#each lines as line, index (index)}
			{@const Icon = ICON[line.kind]}
			<div class="flex items-center gap-2.5 whitespace-nowrap {TONE[line.kind]}">
				<Icon size={15} aria-hidden="true" class="shrink-0" />
				<span class={line.kind === 'command' ? 'font-medium' : ''}>{line.text}</span>
			</div>
		{/each}
	</div>
</div>
