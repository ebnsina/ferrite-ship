<script lang="ts">
	import CtaBand from '$components/marketing/CtaBand.svelte';
	import Faq from '$components/marketing/Faq.svelte';
	import FeatureGrid from '$components/marketing/FeatureGrid.svelte';
	import Hero from '$components/marketing/Hero.svelte';
	import HowItWorks from '$components/marketing/HowItWorks.svelte';
	import Security from '$components/marketing/Security.svelte';
	import ServiceGrid from '$components/marketing/ServiceGrid.svelte';
	import StatsBar from '$components/marketing/StatsBar.svelte';
	import UseCases from '$components/marketing/UseCases.svelte';
	import Seo from '$components/Seo.svelte';
</script>

<Seo
	bare
	title="Ferrite Ship · Run your server from your browser"
	description="Connect a fresh Ubuntu server, get it set up safely in minutes, then manage everything — files, apps, updates, storage — from one dashboard. No command line needed."
	path="/"
/>

<Hero />
<StatsBar />
<FeatureGrid />
<HowItWorks />
<UseCases />
<ServiceGrid />
<Security />
<Faq />
<CtaBand />
