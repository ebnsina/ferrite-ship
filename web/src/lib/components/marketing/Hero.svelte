<script lang="ts">
	import { ButtonLink } from '$components/ui';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import PlayCircle from '@lucide/svelte/icons/play-circle';
	import TerminalPreview from './TerminalPreview.svelte';

	const assurances = ['Works with any Ubuntu server', 'No command line needed', 'Set up in a minute'];
</script>

<section class="relative overflow-hidden">
	<!-- Accent wash sits behind the header too, which is why the header has no border. -->
	<div
		class="pointer-events-none absolute inset-x-0 -top-56 h-[34rem] opacity-[0.16] blur-3xl"
		style="background: radial-gradient(45% 55% at 50% 50%, var(--ui-accent), transparent 70%)"
		aria-hidden="true"
	></div>

	<div class="relative mx-auto max-w-6xl px-5 pt-14 pb-16 sm:pt-20">
		<div class="mx-auto max-w-3xl text-center">
			<h1 class="text-content text-4xl font-semibold tracking-tight sm:text-5xl md:text-6xl">
				Your server, run from
				<span class="text-accent">your browser</span>.
			</h1>

			<p class="text-content-muted mx-auto mt-6 max-w-2xl text-base sm:text-lg">
				Connect a new server and we will get it set up safely in minutes. After that you can manage
				everything — your files, your apps, updates, storage — from one page, without ever typing a
				command.
			</p>

			<div class="mt-9 flex flex-wrap items-center justify-center gap-3">
				<ButtonLink href="/dashboard" size="lg">
					Get started free
					<ArrowRight size={17} aria-hidden="true" />
				</ButtonLink>
				<ButtonLink href="#how-it-works" variant="secondary" size="lg">
					<PlayCircle size={17} aria-hidden="true" />
					See how it works
				</ButtonLink>
			</div>

			<ul class="mt-8 flex flex-wrap items-center justify-center gap-x-6 gap-y-2">
				{#each assurances as assurance (assurance)}
					<li class="text-content-subtle flex items-center gap-1.5 text-xs">
						<CircleCheck size={14} aria-hidden="true" class="text-ok" />
						{assurance}
					</li>
				{/each}
			</ul>
		</div>

		<div class="mx-auto mt-16 max-w-2xl">
			<TerminalPreview />
		</div>
	</div>
</section>
