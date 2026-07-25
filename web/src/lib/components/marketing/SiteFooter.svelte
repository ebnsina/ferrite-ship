<script lang="ts">
	import Wordmark from '$components/brand/Wordmark.svelte';
	import { formatNumber } from '$utils/format';

	const year = formatNumber(new Date().getFullYear(), { useGrouping: false });

	const columns = [
		{
			heading: 'Product',
			links: [
				{ label: 'What you get', href: '#features' },
				{ label: 'Tools', href: '#services' },
				{ label: 'How it works', href: '#how-it-works' }
			]
		},
		{
			heading: 'Your account',
			links: [
				{ label: 'Dashboard', href: '/dashboard' },
				{ label: 'Servers', href: '/dashboard/servers' }
			]
		}
	];
</script>

<footer class="border-border/70 border-t">
	<div class="mx-auto flex max-w-6xl flex-col gap-10 px-5 py-14 sm:flex-row sm:justify-between">
		<div class="max-w-xs">
			<Wordmark />
			<p class="text-content-muted mt-4 text-sm leading-relaxed">
				Set up and look after your own server, without needing to know how any of it works
				underneath.
			</p>
		</div>

		<div class="flex gap-16">
			{#each columns as column (column.heading)}
				<div>
					<h2 class="text-content text-xs font-medium tracking-[0.12em] uppercase">
						{column.heading}
					</h2>
					<ul class="mt-4 space-y-2.5">
						{#each column.links as link (link.href)}
							<li>
								<a
									href={link.href}
									class="text-content-muted hover:text-content text-sm transition-colors duration-150"
								>
									{link.label}
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</div>
	</div>

	<div class="border-border/70 border-t">
		<p class="text-content-subtle mx-auto max-w-6xl px-5 py-6 text-xs">
			© {year} ferrite-ship
		</p>
	</div>
</footer>
