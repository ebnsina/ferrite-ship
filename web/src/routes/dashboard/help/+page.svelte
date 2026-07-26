<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import ComingSoon from '$components/dashboard/ComingSoon.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import LifeBuoy from '@lucide/svelte/icons/life-buoy';
</script>

<Seo title="Help" description="Guides, answers and a way to reach a person." noindex />

<DashboardTopbar
	crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Help' }]}
	unread={2}
/>

<div class="space-y-6 px-6 py-8">
	<SectionHeader title="Help" description="Guides, answers, and a way to reach a person." />

	<ComingSoon
		icon={LifeBuoy}
		title="Not ready just yet"
		description="Step-by-step guides and a contact form are on the way. Until then, email us and we will help."
	/>
</div>
