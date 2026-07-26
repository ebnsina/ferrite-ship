<script lang="ts">
	import CtaBand from '$components/marketing/CtaBand.svelte';
	import Faq from '$components/marketing/Faq.svelte';
	import FeatureGrid from '$components/marketing/FeatureGrid.svelte';
	import Hero from '$components/marketing/Hero.svelte';
	import HowItWorks from '$components/marketing/HowItWorks.svelte';
	import Pricing from '$components/marketing/Pricing.svelte';
	import Security from '$components/marketing/Security.svelte';
	import ServiceGrid from '$components/marketing/ServiceGrid.svelte';
	import StatsBar from '$components/marketing/StatsBar.svelte';
	import Testimonials from '$components/marketing/Testimonials.svelte';
	import UseCases from '$components/marketing/UseCases.svelte';
</script>

<svelte:head>
	<title>Ferrite Ship · Run your server from your browser</title>
	<meta
		name="description"
		content="Connect a fresh Ubuntu server, get it set up safely in minutes, then manage everything — files, apps, updates, storage — from one dashboard. No command line needed."
	/>
</svelte:head>

<Hero />
<StatsBar />
<FeatureGrid />
<HowItWorks />
<UseCases />
<ServiceGrid />
<Security />
<Pricing />
<Testimonials />
<Faq />
<CtaBand />
