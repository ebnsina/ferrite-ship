<script lang="ts">
	import { formatNumber } from '$utils/format';

	interface Props {
		count: number;
	}

	let { count }: Props = $props();
</script>

<span
	class="bg-surface-sunken text-content-muted inline-flex min-w-5 items-center justify-center rounded-pill px-1.5 py-0.5 text-xs font-medium"
	data-numeric
>
	{formatNumber(count)}
</span>
