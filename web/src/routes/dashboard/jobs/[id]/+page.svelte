<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import JobTimeline from '$components/dashboard/JobTimeline.svelte';
	import { ButtonLink } from '$components/ui';
	import { TONE_CLASSES, type Tone } from '$components/ui/tone';
	import { createJobStream } from '$lib/jobs/job-stream.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	const jobId = page.params.id ?? '';
	const stream = createJobStream(jobId);

	const banner = $derived.by(() => {
		if (stream.state === 'error') {
			return {
				tone: 'error' as Tone,
				icon: TriangleAlert,
				spin: false,
				title: 'Lost contact with the control plane',
				detail: 'The job may still be running. Reload to pick the log back up.'
			};
		}
		if (stream.state !== 'done') {
			return {
				tone: 'info' as Tone,
				icon: LoaderCircle,
				spin: true,
				title: 'Working on it',
				detail: 'Each step is checked first, and only changed if it needs to be.'
			};
		}
		if (stream.outcome === 'failed') {
			return {
				tone: 'error' as Tone,
				icon: CircleX,
				spin: false,
				title: 'Something did not work',
				detail: stream.summary ?? 'Open the failed step below to see what happened.'
			};
		}
		return {
			tone: 'ok' as Tone,
			icon: CircleCheck,
			spin: false,
			title: 'All done',
			detail: stream.summary ?? 'Everything finished successfully.'
		};
	});

	const BannerIcon = $derived(banner.icon);
</script>

<Seo title="Setting up" description="Watch a setup run as it happens." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'History', href: '/dashboard/activity' },
		{ label: 'Job' }
	]}
/>

<div class="mx-auto max-w-3xl space-y-6 px-6 py-8">
	<div class="flex items-start gap-4 rounded-card p-5 {TONE_CLASSES[banner.tone].soft}">
		<BannerIcon
			size={20}
			aria-hidden="true"
			class={banner.spin ? 'mt-0.5 shrink-0 animate-spin' : 'mt-0.5 shrink-0'}
		/>
		<div class="min-w-0">
			<h1 class="text-sm font-medium">{banner.title}</h1>
			<p class="text-content-muted mt-1 text-sm">{banner.detail}</p>
		</div>
	</div>

	<JobTimeline {stream} />

	<div class="flex flex-wrap gap-3">
		<ButtonLink href="/dashboard/servers" variant="secondary" size="sm">
			<ArrowLeft size={15} aria-hidden="true" />
			Back to servers
		</ButtonLink>
		{#if stream.state === 'done'}
			<ButtonLink href="/dashboard" size="sm">Go to overview</ButtonLink>
		{/if}
	</div>
</div>
