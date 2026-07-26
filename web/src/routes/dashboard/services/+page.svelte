<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import ComingSoon from '$components/dashboard/ComingSoon.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Blocks from '@lucide/svelte/icons/blocks';
</script>

<Seo title="Tools" description="Databases, caches, storage and streaming." noindex />

<DashboardTopbar
	crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Tools' }]}
	unread={2}
/>

<div class="space-y-6 px-6 py-8">
	<SectionHeader
		title="Tools"
		description="Databases, caches, storage and streaming you can add to any server."
	/>

	<ComingSoon
		icon={Blocks}
		title="Not ready just yet"
		description="This is where you will pick a tool and have it running on a server in one click. We are building it now."
	/>
</div>
