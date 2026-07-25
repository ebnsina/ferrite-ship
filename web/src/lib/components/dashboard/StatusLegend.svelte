<script lang="ts">
	import { TONE_CLASSES, type Tone } from '$components/ui/tone';

	/** Spells out what the bar colours mean, so nobody has to guess. */
	const entries: { tone: Tone; meaning: string }[] = [
		{ tone: 'ok', meaning: 'Plenty of room' },
		{ tone: 'warn', meaning: 'Getting full' },
		{ tone: 'error', meaning: 'Needs attention' }
	];
</script>

<ul class="flex flex-wrap items-center gap-x-5 gap-y-2">
	{#each entries as entry (entry.tone)}
		<li class="text-content-subtle flex items-center gap-2 text-xs">
			<span class="size-2 rounded-pill {TONE_CLASSES[entry.tone].fill}" aria-hidden="true"></span>
			{entry.meaning}
		</li>
	{/each}
</ul>
