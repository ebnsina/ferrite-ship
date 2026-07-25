<script lang="ts">
	import type { LucideProps } from '@lucide/svelte';
	import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
	import Coffee from '@lucide/svelte/icons/coffee';
	import MonitorSmartphone from '@lucide/svelte/icons/monitor-smartphone';
	import type { Component } from 'svelte';
	import SectionHeading from './SectionHeading.svelte';

	interface Step {
		number: string;
		title: string;
		description: string;
		icon: Component<LucideProps>;
	}

	const steps: Step[] = [
		{
			number: '01',
			title: 'Paste one line',
			description:
				'Copy the setup line from your dashboard and run it once on your new server. That is the only time you touch a command.',
			icon: ClipboardCheck
		},
		{
			number: '02',
			title: 'Let it settle in',
			description:
				'We tidy up the security settings, switch on automatic updates, and check everything is healthy. Run it again any time — nothing breaks.',
			icon: Coffee
		},
		{
			number: '03',
			title: 'Run it from anywhere',
			description:
				'Open your dashboard in any browser and your server is right there: files, apps, updates and alerts, all in one place.',
			icon: MonitorSmartphone
		}
	];
</script>

<section id="how-it-works" class="mx-auto max-w-6xl scroll-mt-20 px-5 py-20">
	<SectionHeading
		eyebrow="How it works"
		title="Three steps, and nothing to learn"
		description="You will not need to open a terminal on your own computer, or leave anything new exposed to the internet."
	/>

	<ol class="mt-14 grid gap-5 md:grid-cols-3">
		{#each steps as step (step.number)}
			{@const Icon = step.icon}
			<li class="border-border bg-surface rounded-card border p-7">
				<div class="flex items-center gap-3">
					<span class="bg-accent-soft text-accent flex size-10 items-center justify-center rounded-tile">
						<Icon size={18} aria-hidden="true" />
					</span>
					<span class="text-content-subtle text-sm font-semibold" data-numeric>{step.number}</span>
				</div>
				<h3 class="text-content mt-5 text-base font-medium">{step.title}</h3>
				<p class="text-content-muted mt-2 text-sm leading-relaxed">{step.description}</p>
			</li>
		{/each}
	</ol>
</section>
