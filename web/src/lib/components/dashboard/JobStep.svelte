<script lang="ts">
	import { TONE_CLASSES } from '$components/ui/tone';
	import { STEP_OUTCOME } from '$lib/domain/status';
	import type { LogLevel, StepRun } from '$types/job';
	import { cn } from '$utils/cn';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';
	import Minus from '@lucide/svelte/icons/minus';

	interface Props {
		step: StepRun;
		/** Open by default while the step is still running or if it failed. */
		defaultOpen?: boolean;
		/**
		 * Whether the job is still going. A step with no outcome means
		 * "in progress" only while that is true — once the job has ended, a step
		 * that never reported one is finished, not running, and spinning at
		 * someone forever makes a completed job look stuck.
		 */
		live?: boolean;
	}

	let { step, defaultOpen = false, live = true }: Props = $props();

	// Steps open while running or on failure, and collapse once they settle —
	// unless the reader has expressed a preference by clicking.
	let override = $state<boolean | null>(null);
	const open = $derived(override ?? defaultOpen);

	const presentation = $derived(step.outcome ? STEP_OUTCOME[step.outcome] : null);
	const Icon = $derived(presentation?.icon);

	// Command lines are the useful skeleton of a log; output is supporting detail.
	const LEVEL_CLASS: Record<LogLevel, string> = {
		command: 'text-content',
		output: 'text-content-subtle',
		info: 'text-content-muted',
		changed: 'text-ok',
		skipped: 'text-info',
		error: 'text-error'
	};

	const PREFIX: Record<LogLevel, string> = {
		command: '$',
		output: ' ',
		info: '·',
		changed: '✓',
		skipped: '–',
		error: '!'
	};
</script>

<div class="border-border/70 border-b last:border-b-0">
	<button
		type="button"
		onclick={() => (override = !open)}
		aria-expanded={open}
		class="hover:bg-surface-sunken flex w-full items-center gap-3 px-5 py-3 text-left transition-colors duration-150"
	>
		<span
			class={cn(
				'shrink-0',
				presentation ? TONE_CLASSES[presentation.tone].text : live ? 'text-info' : 'text-content-subtle'
			)}
		>
			{#if Icon}
				<Icon size={16} aria-hidden="true" />
			{:else if live}
				<LoaderCircle size={16} aria-hidden="true" class="animate-spin" />
			{:else}
				<Minus size={16} aria-hidden="true" />
			{/if}
		</span>

		<span class="text-content min-w-0 flex-1 truncate text-sm">{step.title}</span>

		{#if presentation}
			<span class={cn('text-xs font-medium', TONE_CLASSES[presentation.tone].text)}>
				{presentation.label}
			</span>
		{:else if live}
			<span class="text-content-subtle text-xs">Running…</span>
		{/if}

		{#if step.lines.length > 0}
			<ChevronRight
				size={15}
				aria-hidden="true"
				class={cn(
					'text-content-subtle shrink-0 transition-transform duration-150',
					open && 'rotate-90'
				)}
			/>
		{:else}
			<span class="size-[15px] shrink-0" aria-hidden="true"></span>
		{/if}
	</button>

	{#if open && step.lines.length > 0}
		<div class="bg-surface-sunken border-border/70 border-t px-5 py-3">
			<pre class="font-machine overflow-x-auto text-xs leading-relaxed"><code
					>{#each step.lines as line (line.seq)}<span class={LEVEL_CLASS[line.level]}
							>{PREFIX[line.level]} {line.message}</span
						>{'\n'}{/each}</code
				></pre>
		</div>
	{/if}
</div>
