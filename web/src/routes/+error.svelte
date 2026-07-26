<script lang="ts">
	import { page } from '$app/state';
	import Wordmark from '$components/brand/Wordmark.svelte';
	import { ButtonLink } from '$components/ui';
	import { codeForStatus, describe } from '$lib/errors';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
	import { formatNumber } from '$utils/format';

	/**
	 * SvelteKit fills in a terse HTTP reason phrase ("Not Found") when nothing
	 * better exists. Those say less than our own copy, so they are dropped.
	 */
	const GENERIC_MESSAGES = new Set([
		'Not Found',
		'Internal Error',
		'Internal Server Error',
		'Bad Request',
		'Forbidden',
		'Unauthorized'
	]);

	const fallback = $derived(describe(codeForStatus(page.status)));
	const heading = $derived(page.status === 404 ? 'We cannot find that page' : fallback.message);

	const detail = $derived.by(() => {
		const message = page.error?.message;
		if (!message || GENERIC_MESSAGES.has(message)) return null;
		return message === heading ? null : message;
	});
</script>

<svelte:head>
	<title>{page.status} · Ferrite Ship</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="flex min-h-dvh flex-col">
	<header class="mx-auto w-full max-w-6xl px-5 py-6">
		<Wordmark />
	</header>

	<main class="flex flex-1 items-center justify-center px-5 py-16">
		<div class="w-full max-w-md text-center">
			<p class="text-accent font-machine text-5xl font-semibold" data-numeric>
				{formatNumber(page.status, { useGrouping: false })}
			</p>

			<h1 class="text-content mt-6 text-2xl font-semibold tracking-tight">{heading}</h1>

			{#if detail}
				<p class="text-content-muted mt-3 text-sm">{detail}</p>
			{/if}

			<p class="text-content-muted mt-3 text-sm">{fallback.action}</p>

			{#if page.error?.requestId}
				<p class="text-content-subtle font-machine mt-6 text-xs">
					Reference: {page.error.requestId}
				</p>
			{/if}

			<div class="mt-9 flex flex-wrap justify-center gap-3">
				<ButtonLink href="/">
					<ArrowLeft size={16} aria-hidden="true" />
					Back to home
				</ButtonLink>
				<ButtonLink href="/dashboard" variant="secondary">
					<LayoutDashboard size={16} aria-hidden="true" />
					Go to dashboard
				</ButtonLink>
			</div>
		</div>
	</main>
</div>
