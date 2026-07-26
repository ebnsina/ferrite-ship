<script lang="ts">
	import QueryLibrary from '$components/dashboard/QueryLibrary.svelte';
	import { Button } from '$components/ui';
	import { toolsClient, type QueryResult, type Tool } from '$lib/data/tools';
	import { toAppError, type AppError } from '$lib/errors';
	import { formatNumber } from '$utils/format';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import Play from '@lucide/svelte/icons/play';
	import type { Attachment } from 'svelte/attachments';

	interface Props {
		serverId: string;
		tool: Tool;
	}

	let { serverId, tool }: Props = $props();

	let query = $state('');
	let result = $state<QueryResult | null>(null);
	let error = $state<AppError | null>(null);
	let running = $state(false);

	// The last query that produced what is on screen, so the results header can
	// say what they are the results *of* after the editor has been edited.
	let ran = $state('');

	async function run() {
		if (running || !query.trim()) return;

		running = true;
		error = null;

		try {
			result = await toolsClient.query(serverId, tool.id, query);
			ran = query.trim();
		} catch (cause) {
			error = toAppError(cause);
			result = null;
		} finally {
			running = false;
		}
	}

	/*
	 * Cmd/Ctrl+Enter runs, because a console where the only way to submit is to
	 * reach for the mouse is a console nobody uses twice. Enter alone inserts a
	 * newline: queries are frequently more than one line, and a stray Return
	 * running a DELETE would be a very bad default.
	 */
	const submitOnMeta: Attachment<HTMLTextAreaElement> = (node) => {
		const onKeydown = (event: KeyboardEvent) => {
			if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
				event.preventDefault();
				void run();
			}
		};
		node.addEventListener('keydown', onKeydown);
		return () => node.removeEventListener('keydown', onKeydown);
	};
</script>

<div class="space-y-4">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h2 class="text-content text-sm font-medium">Run {tool.consoleLanguage ?? 'a query'}</h2>
			<p class="text-content-muted mt-0.5 text-sm">
				This runs against {tool.name} on this server, as its owner. Anything you can do here, you can
				undo only if you wrote it down.
			</p>
		</div>

		<Button onclick={run} disabled={running || !query.trim()}>
			<Play size={15} aria-hidden="true" />
			{running ? 'Running…' : 'Run'}
		</Button>
	</div>

	<QueryLibrary {serverId} {tool} current={query} onPick={(picked) => (query = picked)} />

	<textarea
		bind:value={query}
		{@attach submitOnMeta}
		rows="6"
		spellcheck="false"
		autocomplete="off"
		autocapitalize="off"
		placeholder={tool.consolePlaceholder}
		aria-label="{tool.consoleLanguage ?? 'Query'} to run against {tool.name}"
		class="border-border bg-surface-sunken text-content placeholder:text-content-subtle font-machine rounded-field focus-visible:border-accent-border w-full resize-y border px-3.5 py-3 text-xs leading-relaxed outline-none transition-colors duration-150"
	></textarea>

	<p class="text-content-subtle text-xs">
		Press <kbd class="border-border rounded-[0.375rem] border px-1 py-0.5">⌘</kbd> +
		<kbd class="border-border rounded-[0.375rem] border px-1 py-0.5">Enter</kbd> to run.
	</p>

	{#if error}
		<div class="border-error/40 bg-error-soft rounded-card-sm flex gap-3 border p-4">
			<CircleAlert size={17} aria-hidden="true" class="text-error mt-0.5 shrink-0" />
			<div>
				<p class="text-error text-sm">{error.message}</p>
				{#if error.action}
					<p class="text-content-muted mt-1 text-xs">{error.action}</p>
				{/if}
			</div>
		</div>
	{:else if result?.failed}
		<!-- The database's own words. A syntax error tells you where the problem
		     is far better than any sentence we could write about it. -->
		<div class="border-error/40 bg-error-soft rounded-card-sm border p-4">
			<p class="text-error text-xs font-medium">{tool.name} would not run that</p>
			<pre
				class="text-error font-machine mt-2 overflow-x-auto text-xs whitespace-pre-wrap">{result.message}</pre>
		</div>
	{:else if result}
		<div class="space-y-2">
			<p class="text-content-subtle text-xs">
				{#if result.rows.length === 0}
					Done. Nothing to show — that statement returns no rows.
				{:else}
					{formatNumber(result.rows.length)}
					{result.rows.length === 1 ? 'row' : 'rows'}
				{/if}
				· took {formatNumber(result.tookMs)}ms
				{#if result.truncated}
					· <span class="text-warn">only the first {formatNumber(result.rows.length)} are shown</span>
				{/if}
			</p>

			{#if result.rows.length > 0}
				<!-- The table scrolls inside its own box rather than widening the
				     page: a query with forty columns is normal and should not push
				     the rest of the dashboard sideways. -->
				<div class="border-border rounded-card-sm overflow-x-auto border">
					<table class="w-full text-left text-xs">
						<thead class="bg-surface-sunken text-content-muted">
							<tr>
								{#each result.columns as column, index (index)}
									<th class="font-machine px-3 py-2 font-medium whitespace-nowrap">{column}</th>
								{/each}
							</tr>
						</thead>
						<tbody class="divide-border/70 divide-y">
							{#each result.rows as row, rowIndex (rowIndex)}
								<tr class="hover:bg-surface-sunken/60">
									{#each row as cell, cellIndex (cellIndex)}
										<td class="text-content font-machine px-3 py-2 whitespace-nowrap">
											{#if cell === ''}
												<span class="text-content-subtle italic">empty</span>
											{:else}
												{cell}
											{/if}
										</td>
									{/each}
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}

			{#if ran}
				<p class="text-content-subtle font-machine truncate text-xs" title={ran}>{ran}</p>
			{/if}
		</div>
	{/if}
</div>
