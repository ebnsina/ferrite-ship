<script lang="ts">
	import Wordmark from '$components/brand/Wordmark.svelte';
	import { ButtonLink, ThemeToggle } from '$components/ui';
	import { marketingNav } from '$lib/content/marketing-nav';
	import { siteTheme } from '$lib/theme/theme.svelte';
	import Menu from '@lucide/svelte/icons/menu';
	import X from '@lucide/svelte/icons/x';

	let open = $state(false);
</script>

<!-- No bottom border: the header sits over the hero's accent wash and blends. -->
<header
	class="supports-[backdrop-filter]:bg-canvas/50 bg-canvas/85 sticky top-0 z-40 backdrop-blur-xl"
>
	<div class="mx-auto grid h-16 max-w-6xl grid-cols-[auto_1fr_auto] items-center gap-6 px-5">
		<Wordmark size="sm" />

		<nav aria-label="Primary" class="hidden justify-center md:flex">
			<ul class="flex items-center gap-1">
				{#each marketingNav as item (item.href)}
					<li>
						<a
							href={item.href}
							class="text-content-muted hover:text-content rounded-tile px-3.5 py-2 text-sm transition-colors duration-150"
						>
							{item.label}
						</a>
					</li>
				{/each}
			</ul>
		</nav>

		<div class="flex items-center justify-end gap-2">
			<ThemeToggle theme={siteTheme.value} onToggle={() => siteTheme.toggle()} />

			<a
				href="/dashboard"
				class="text-content-muted hover:text-content hidden px-3 py-2 text-sm font-medium transition-colors duration-150 sm:block"
			>
				Sign in
			</a>

			<ButtonLink href="/dashboard" size="sm" class="hidden sm:inline-flex">Get started</ButtonLink>

			<button
				type="button"
				onclick={() => (open = !open)}
				class="text-content-muted hover:bg-surface-raised hover:text-content inline-flex size-9 items-center justify-center rounded-tile transition-colors duration-150 md:hidden"
				aria-expanded={open}
				aria-label={open ? 'Close menu' : 'Open menu'}
			>
				{#if open}
					<X size={18} aria-hidden="true" />
				{:else}
					<Menu size={18} aria-hidden="true" />
				{/if}
			</button>
		</div>
	</div>

	{#if open}
		<div class="border-border/70 bg-canvas border-t md:hidden">
			<nav aria-label="Primary" class="mx-auto max-w-6xl px-5 py-4">
				<ul class="space-y-1">
					{#each marketingNav as item (item.href)}
						<li>
							<a
								href={item.href}
								onclick={() => (open = false)}
								class="text-content-muted hover:bg-surface-raised hover:text-content block rounded-tile px-3 py-2.5 text-sm transition-colors duration-150"
							>
								{item.label}
							</a>
						</li>
					{/each}
				</ul>

				<div class="mt-4 flex flex-col gap-2 sm:hidden">
					<ButtonLink href="/dashboard" variant="secondary" size="sm">Sign in</ButtonLink>
					<ButtonLink href="/dashboard" size="sm">Get started</ButtonLink>
				</div>
			</nav>
		</div>
	{/if}
</header>
