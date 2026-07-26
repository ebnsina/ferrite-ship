<script lang="ts">
	import Wordmark from '$components/brand/Wordmark.svelte';
	import { Button } from '$components/ui';
	import { footerColumns } from '$lib/content/footer';
	import { formatNumber } from '$utils/format';
	import SocialLinks from './SocialLinks.svelte';

	const year = formatNumber(new Date().getFullYear(), { useGrouping: false });

	const legal = [
		{ label: 'Privacy', href: '#' },
		{ label: 'Terms', href: '#' },
		{ label: 'Cookies', href: '#' }
	];
</script>

<footer class="border-border/70 border-t">
	<div class="mx-auto max-w-6xl px-5">
		<!-- Newsletter -->
		<div
			class="border-border/70 flex flex-col gap-6 border-b py-12 md:flex-row md:items-center md:justify-between"
		>
			<div class="max-w-md">
				<h2 class="text-content text-base font-medium">Keep up with what we ship</h2>
				<p class="text-content-muted mt-1.5 text-sm">
					An occasional email when something useful lands. No noise, and you can leave whenever.
				</p>
			</div>

			<form class="flex w-full max-w-sm gap-2" onsubmit={(event) => event.preventDefault()}>
				<label for="footer-email" class="sr-only">Email address</label>
				<input
					id="footer-email"
					type="email"
					required
					placeholder="you@example.com"
					class="border-border bg-surface-sunken text-content placeholder:text-content-subtle focus-visible:border-accent-border h-10 min-w-0 flex-1 rounded-field border px-3.5 text-sm outline-none transition-colors duration-150"
				/>
				<Button type="submit">Subscribe</Button>
			</form>
		</div>

		<!-- Link columns -->
		<div class="grid gap-10 py-14 sm:grid-cols-2 lg:grid-cols-6">
			<div class="lg:col-span-2">
				<Wordmark />
				<p class="text-content-muted mt-4 max-w-xs text-sm leading-relaxed">
					Set up and look after your own server, without needing to know how any of it works
					underneath.
				</p>
				<div class="mt-5">
					<SocialLinks />
				</div>
			</div>

			{#each footerColumns as column (column.heading)}
				<div>
					<h2 class="text-content text-xs font-medium tracking-[0.12em] uppercase">
						{column.heading}
					</h2>
					<ul class="mt-4 space-y-2.5">
						{#each column.links as link (link.label)}
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

	<!-- Bottom bar -->
	<div class="border-border/70 border-t">
		<div
			class="mx-auto flex max-w-6xl flex-col-reverse items-center gap-4 px-5 py-6 sm:flex-row sm:justify-between"
		>
			<p class="text-content-subtle text-xs">© {year} Ferrite Ship. All rights reserved.</p>

			<div class="flex flex-wrap items-center justify-center gap-x-5 gap-y-2">
				<span class="text-content-subtle flex items-center gap-2 text-xs">
					<span class="bg-ok size-1.5 rounded-pill" aria-hidden="true"></span>
					All systems normal
				</span>

				{#each legal as item (item.label)}
					<a
						href={item.href}
						class="text-content-subtle hover:text-content text-xs transition-colors duration-150"
					>
						{item.label}
					</a>
				{/each}
			</div>
		</div>
	</div>
</footer>
