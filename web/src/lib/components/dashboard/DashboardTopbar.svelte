<script lang="ts">
	import { ThemeToggle } from '$components/ui';
	import { page } from '$app/state';
	import { dashboardTheme } from '$lib/theme/theme.svelte';
	import { serverIdFromPath, serverSectionHref, serverSections } from '$lib/domain/server-nav';
	import type { Crumb } from '$types/breadcrumb';
	import Breadcrumb from './Breadcrumb.svelte';

	/*
	 * The notifications bell is gone, along with the `unread={2}` that four
	 * pages passed it. There is no notifications feature, so the red dot was
	 * announcing two things that did not exist and the button opened nothing.
	 *
	 * The search box has gone the same way, and for the same reason: it had no
	 * handler at all. A box you can type into that never answers is worse than
	 * no box, because it reads as a search that found nothing. The servers list
	 * has a real one that filters as you type.
	 *
	 * Both come back when there is something behind them.
	 */
	interface Props {
		crumbs: Crumb[];
	}

	let { crumbs }: Props = $props();

	/*
	 * The server's other sections are attached here rather than by each page.
	 *
	 * Every page inside a server already puts that server in its trail, so the
	 * one place that knows about all of them can add the sideways links once.
	 * Asking twelve pages to pass the same list is how one of them ends up
	 * without it.
	 */
	const serverId = $derived(serverIdFromPath(page.url.pathname));

	const enriched = $derived.by(() => {
		const id = serverId;
		if (!id) return crumbs;

		const root = `/dashboard/servers/${id}`;
		const siblings = serverSections.map((section) => ({
			label: section.label,
			href: serverSectionHref(id, section)
		}));

		// Exactly one crumb carries the menu. Attaching it wherever it fits gave
		// two identical dropdowns side by side — once on the server's name and
		// again on the section, which reads as two different menus until you
		// open both.
		const anchor = crumbs.findIndex((crumb) => crumb.href === root);
		const target = anchor === -1 ? crumbs.length - 1 : anchor;

		return crumbs.map((crumb, index) =>
			index === target ? { ...crumb, siblings } : crumb
		);
	});
</script>

<header class="border-border/70 flex h-16 items-center gap-4 border-b px-6">
	<Breadcrumb items={enriched} />

	<div class="ml-auto flex items-center gap-2">
		<ThemeToggle theme={dashboardTheme.value} onToggle={() => dashboardTheme.toggle()} />
	</div>
</header>
