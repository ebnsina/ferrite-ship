<script lang="ts">
	import ComingSoon from '$components/dashboard/ComingSoon.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Settings from '@lucide/svelte/icons/settings';
</script>

<svelte:head>
	<title>Settings · ferrite-ship</title>
</svelte:head>

<DashboardTopbar
	crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Settings' }]}
	unread={2}
/>

<div class="space-y-6 px-6 py-8">
	<SectionHeader title="Settings" description="Your account, your team, and how you get notified." />

	<ComingSoon
		icon={Settings}
		title="Not ready just yet"
		description="Account details, teammates and notification preferences will live here."
	/>
</div>
