<script lang="ts">
	import { socialLinks } from '$lib/content/social';
	import Mail from '@lucide/svelte/icons/mail';

	// Lucide no longer ships brand marks, so GitHub and X are inline paths.
	const GITHUB_PATH =
		'M12 .5C5.73.5.9 5.4.9 11.75c0 4.98 3.16 9.2 7.55 10.69.55.1.75-.24.75-.54v-1.9c-3.07.68-3.72-1.33-3.72-1.33-.5-1.3-1.23-1.65-1.23-1.65-1-.7.08-.68.08-.68 1.1.08 1.69 1.15 1.69 1.15.99 1.72 2.59 1.22 3.22.94.1-.73.39-1.22.7-1.5-2.45-.28-5.03-1.24-5.03-5.55 0-1.23.43-2.23 1.13-3.02-.11-.28-.49-1.42.11-2.96 0 0 .93-.3 3.04 1.15a10.4 10.4 0 0 1 5.54 0c2.1-1.45 3.03-1.15 3.03-1.15.6 1.54.22 2.68.11 2.96.7.79 1.13 1.79 1.13 3.02 0 4.32-2.59 5.27-5.05 5.54.4.35.75 1.03.75 2.08v3.09c0 .3.2.65.76.54a11.28 11.28 0 0 0 7.54-10.69C23.1 5.4 18.27.5 12 .5Z';

	const X_PATH =
		'M18.24 2.25h3.31l-7.23 8.26 8.5 11.24h-6.65l-5.21-6.82-5.97 6.82H1.68l7.73-8.84L1.25 2.25h6.82l4.71 6.23 5.46-6.23Zm-1.16 17.52h1.83L7.05 4.13H5.08l12 15.64Z';

	const linkClass =
		'text-content-muted hover:bg-surface-raised hover:text-content inline-flex size-9 items-center justify-center rounded-tile transition-colors duration-150';
</script>

{#if socialLinks.length > 0}
	<ul class="flex items-center gap-1">
		{#each socialLinks as link (link.kind)}
			<li>
				<a
					href={link.href}
					class={linkClass}
					aria-label={link.label}
					rel={link.kind === 'email' ? undefined : 'noreferrer'}
					target={link.kind === 'email' ? undefined : '_blank'}
				>
					{#if link.kind === 'github'}
						<svg width="17" height="17" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
							<path d={GITHUB_PATH} />
						</svg>
					{:else if link.kind === 'x'}
						<svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
							<path d={X_PATH} />
						</svg>
					{:else}
						<Mail size={17} aria-hidden="true" />
					{/if}
				</a>
			</li>
		{/each}
	</ul>
{/if}
