<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import ComingSoon from '$components/dashboard/ComingSoon.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Settings from '@lucide/svelte/icons/settings';
</script>

<Seo title="Settings" description="Your account, your team, and notifications." noindex />

<DashboardTopbar
	crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Settings' }]}
/>

<div class="space-y-6 px-6 py-8">
	<SectionHeader title="Settings" description="Your account, your team, and how you get notified." />

	<ComingSoon
		icon={Settings}
		title="Not ready just yet"
		description="Account details, teammates and notification preferences will live here."
	/>
</div>
