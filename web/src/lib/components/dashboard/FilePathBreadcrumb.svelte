<script lang="ts">
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import HardDrive from '@lucide/svelte/icons/hard-drive';

	interface Props {
		/** Absolute directory path, e.g. /etc/systemd/system. */
		path: string;
		onNavigate: (path: string) => void;
	}

	let { path, onNavigate }: Props = $props();

	interface Crumb {
		label: string;
		path: string;
	}

	const crumbs = $derived.by<Crumb[]>(() => {
		let built = '';
		return path
			.split('/')
			.filter(Boolean)
			.map((segment) => {
				built += `/${segment}`;
				return { label: segment, path: built };
			});
	});

	// Deep paths would otherwise push the trail off the page. Keep the first
	// segment and the last three — the ones people navigate by — and collapse
	// the middle behind a control that jumps to where it was cut.
	const VISIBLE_TAIL = 3;
	const collapsed = $derived(crumbs.length > VISIBLE_TAIL + 2);

	const head = $derived(collapsed ? crumbs.slice(0, 1) : crumbs);
	const tail = $derived(collapsed ? crumbs.slice(-VISIBLE_TAIL) : []);
	const hidden = $derived(collapsed ? crumbs.slice(1, -VISIBLE_TAIL) : []);

	const linkClass =
		'text-content-muted hover:text-content hover:bg-surface-sunken font-machine rounded-tile px-2 py-1 text-xs transition-colors duration-150';
	const currentClass = 'text-content font-machine rounded-tile px-2 py-1 text-xs font-medium';
</script>

<nav aria-label="Folder path" class="min-w-0">
	<ol class="flex flex-wrap items-center gap-0.5">
		<li>
			<button
				type="button"
				onclick={() => onNavigate('/')}
				class="{path === '/' ? currentClass : linkClass} inline-flex items-center gap-1.5"
				aria-current={path === '/' ? 'page' : undefined}
				aria-label="Root folder"
			>
				<HardDrive size={13} aria-hidden="true" />
				/
			</button>
		</li>

		{#each head as crumb (crumb.path)}
			{@const current = crumb.path === path}
			<li class="flex items-center gap-0.5">
				<ChevronRight size={13} aria-hidden="true" class="text-content-subtle shrink-0" />
				<button
					type="button"
					onclick={() => onNavigate(crumb.path)}
					class={current ? currentClass : linkClass}
					aria-current={current ? 'page' : undefined}
				>
					{crumb.label}
				</button>
			</li>
		{/each}

		{#if collapsed}
			{@const jumpTo = hidden[hidden.length - 1]}
			<li class="flex items-center gap-0.5">
				<ChevronRight size={13} aria-hidden="true" class="text-content-subtle shrink-0" />
				<button
					type="button"
					onclick={() => jumpTo && onNavigate(jumpTo.path)}
					class={linkClass}
					title="Skipped: {hidden.map((crumb) => crumb.label).join('/')}"
				>
					…
				</button>
			</li>
		{/if}

		{#each tail as crumb (crumb.path)}
			{@const current = crumb.path === path}
			<li class="flex items-center gap-0.5">
				<ChevronRight size={13} aria-hidden="true" class="text-content-subtle shrink-0" />
				<button
					type="button"
					onclick={() => onNavigate(crumb.path)}
					class={current ? currentClass : linkClass}
					aria-current={current ? 'page' : undefined}
				>
					{crumb.label}
				</button>
			</li>
		{/each}
	</ol>
</nav>
