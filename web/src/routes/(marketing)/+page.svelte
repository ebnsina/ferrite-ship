<script lang="ts">
	import CtaBand from '$components/marketing/CtaBand.svelte';
	import FeatureGrid from '$components/marketing/FeatureGrid.svelte';
	import Hero from '$components/marketing/Hero.svelte';
	import HowItWorks from '$components/marketing/HowItWorks.svelte';
	import ServiceGrid from '$components/marketing/ServiceGrid.svelte';
</script>

<svelte:head>
	<title>ferrite-ship · Operate your VPS from the browser</title>
	<meta
		name="description"
		content="Connect a fresh Ubuntu VPS, harden it in minutes, then run the whole machine — shell, files, services, packages and disks — from one dashboard."
	/>
</svelte:head>

<Hero />
<FeatureGrid />
<ServiceGrid />
<HowItWorks />
<CtaBand />
