<script lang="ts">
	import { formatDayAndMonth } from '$utils/format';
	import type { Snippet } from 'svelte';

	/**
	 * The top of the overview.
	 *
	 * It used to lead with "3 of 3 of your servers are running fine", which put
	 * the same number the strip below already shows in the largest type on the
	 * page — and said "everything is looking good" on a screen whose whole job
	 * is to let you decide that for yourself. A greeting and the date orient
	 * you without competing with the data.
	 */
	/*
	 * No name in the greeting: an account has an email address and nothing
	 * else, and turning "es@ferrite.co" into "Es" is a guess dressed up as
	 * familiarity. When accounts carry a real name this can use it.
	 */
	interface Props {
		actions?: Snippet;
	}

	let { actions }: Props = $props();

	// Read once per mount rather than kept live: nobody is watching this page
	// at midnight waiting for the word to change, and a ticking clock here
	// would re-render the whole heading every second for nothing.
	const now = new Date();

	const salutation = $derived.by(() => {
		const hour = now.getHours();
		if (hour < 12) return 'Good morning';
		if (hour < 18) return 'Good afternoon';
		return 'Good evening';
	});
</script>

<div class="flex flex-wrap items-end justify-between gap-4">
	<div>
		<h1 class="text-content text-2xl font-semibold tracking-tight sm:text-3xl">{salutation}</h1>
		<p class="text-content-muted mt-1.5 text-sm">
			<!-- datetime carries the machine-readable form, so the pretty string
			     above it is free to be as human as it likes. -->
			<time datetime={now.toISOString().slice(0, 10)}>{formatDayAndMonth(now)}</time>
		</p>
	</div>

	{#if actions}
		<div class="flex items-center gap-2">{@render actions()}</div>
	{/if}
</div>
