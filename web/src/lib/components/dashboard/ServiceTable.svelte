<script lang="ts">
	import { Button, Card } from '$components/ui';
	import { TONE_CLASSES } from '$components/ui/tone';
	import type { ServiceAction, ServiceUnit } from '$lib/data/services';
	import { describeBoot, describeState } from '$lib/domain/service-state';
	import Lock from '@lucide/svelte/icons/lock';
	import Play from '@lucide/svelte/icons/play';
	import RotateCw from '@lucide/svelte/icons/rotate-cw';
	import ScrollText from '@lucide/svelte/icons/scroll-text';
	import Square from '@lucide/svelte/icons/square';

	interface Props {
		units: ServiceUnit[];
		/** Unit currently being acted on, so its row can show it is busy. */
		busyUnit: string | null;
		onAction: (unit: ServiceUnit, action: ServiceAction) => void;
		onViewLogs: (unit: ServiceUnit) => void;
	}

	let { units, busyUnit, onAction, onViewLogs }: Props = $props();
</script>

<Card padded={false} class="overflow-hidden">
	<div class="overflow-x-auto">
		<table class="w-full min-w-3xl text-sm">
			<caption class="sr-only">Services on this server</caption>
			<thead class="border-border/70 text-content-subtle border-b text-xs">
				<tr>
					<th scope="col" class="px-5 py-2.5 text-left font-medium">Service</th>
					<th scope="col" class="px-3 py-2.5 text-left font-medium">State</th>
					<th scope="col" class="hidden px-3 py-2.5 text-left font-medium lg:table-cell">
						At startup
					</th>
					<th scope="col" class="px-5 py-2.5 text-right font-medium">
						<span class="sr-only">Actions</span>
					</th>
				</tr>
			</thead>

			<tbody class="divide-border/70 divide-y">
				{#each units as unit (unit.name)}
					{@const state = describeState(unit)}
					{@const busy = busyUnit === unit.name}
					{@const running = unit.active === 'active'}

					<tr class="hover:bg-surface-sunken group transition-colors duration-150">
						<td class="max-w-0 px-5 py-2.5">
							<div class="flex items-center gap-2">
								<span class="text-content font-machine truncate text-xs">
									{unit.name.replace(/\.service$/, '')}
								</span>
								{#if unit.protected}
									<Lock
										size={12}
										aria-label="Protected — cannot be stopped from here"
										class="text-content-subtle shrink-0"
									/>
								{/if}
							</div>
							{#if unit.description}
								<p class="text-content-subtle mt-0.5 truncate text-xs">{unit.description}</p>
							{/if}
						</td>

						<td class="px-3 py-2.5">
							<span
								class="inline-flex rounded-pill px-2 py-0.5 text-xs font-medium {TONE_CLASSES[
									state.tone
								].soft}"
							>
								{state.label}
							</span>
						</td>

						<td class="text-content-muted hidden px-3 py-2.5 text-xs lg:table-cell">
							{describeBoot(unit)}
						</td>

						<td class="px-5 py-2.5">
							<div
								class="flex items-center justify-end gap-1 opacity-0 transition-opacity duration-150 group-hover:opacity-100 focus-within:opacity-100"
							>
								<Button
									size="sm"
									variant="ghost"
									onclick={() => onViewLogs(unit)}
									aria-label="See the log for {unit.name}"
								>
									<ScrollText size={13} aria-hidden="true" />
								</Button>

								<Button
									size="sm"
									variant="ghost"
									disabled={busy}
									onclick={() => onAction(unit, 'restart')}
									aria-label="Restart {unit.name}"
								>
									<RotateCw size={13} aria-hidden="true" />
								</Button>

								{#if running}
									<Button
										size="sm"
										variant="ghost"
										disabled={busy || unit.protected}
										onclick={() => onAction(unit, 'stop')}
										aria-label={unit.protected
											? `${unit.name} is protected and cannot be stopped here`
											: `Stop ${unit.name}`}
									>
										<Square size={13} aria-hidden="true" />
									</Button>
								{:else}
									<Button
										size="sm"
										variant="ghost"
										disabled={busy}
										onclick={() => onAction(unit, 'start')}
										aria-label="Start {unit.name}"
									>
										<Play size={13} aria-hidden="true" />
									</Button>
								{/if}
							</div>
						</td>
					</tr>
				{/each}

				{#if units.length === 0}
					<tr>
						<td colspan="4" class="text-content-subtle px-5 py-10 text-center text-sm">
							Nothing matches that search.
						</td>
					</tr>
				{/if}
			</tbody>
		</table>
	</div>
</Card>
