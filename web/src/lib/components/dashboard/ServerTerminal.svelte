<script lang="ts">
	import { env } from '$config/env';
	import type { TerminalStatus } from '$types/terminal';
	import { FitAddon } from '@xterm/addon-fit';
	import { Terminal } from '@xterm/xterm';
	import '@xterm/xterm/css/xterm.css';

	interface Props {
		serverId: string;
		/** Reported upward so the page can show the connection state. */
		onStatusChange?: (status: TerminalStatus) => void;
	}

	let { serverId, onStatusChange }: Props = $props();

	let host = $state<HTMLDivElement | null>(null);

	// Terminals are read as machine output, so this stays dark in both themes —
	// the same convention every other terminal follows.
	const THEME = {
		background: '#161714',
		foreground: '#e8ebe3',
		cursor: '#bede4f',
		selectionBackground: '#33361f',
		black: '#161714',
		red: '#e0685f',
		green: '#8fd18a',
		yellow: '#d9c66a',
		blue: '#7fa8e0',
		magenta: '#c99ad8',
		cyan: '#7fcfc4',
		white: '#e8ebe3'
	};

	$effect(() => {
		if (!host) return;

		const term = new Terminal({
			fontFamily: "'Geist Mono Variable', ui-monospace, SFMono-Regular, Menlo, monospace",
			fontSize: 13,
			lineHeight: 1.4,
			cursorBlink: true,
			theme: THEME,
			// Keeps scrollback usable without letting a chatty command eat memory.
			scrollback: 5000
		});

		const fit = new FitAddon();
		term.loadAddon(fit);
		term.open(host);
		fit.fit();

		const url = new URL(`${env.apiBaseUrl}/v1/servers/${encodeURIComponent(serverId)}/terminal`);
		url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
		url.searchParams.set('cols', String(term.cols));
		url.searchParams.set('rows', String(term.rows));

		onStatusChange?.('connecting');

		const socket = new WebSocket(url);
		socket.binaryType = 'arraybuffer';

		socket.addEventListener('open', () => {
			onStatusChange?.('connected');
			term.focus();
		});

		socket.addEventListener('message', (event) => {
			// Output arrives as bytes so xterm's decoder can handle multi-byte
			// characters split across frames.
			term.write(new Uint8Array(event.data as ArrayBuffer));
		});

		socket.addEventListener('close', () => {
			onStatusChange?.('closed');
			term.write('\r\n\x1b[38;5;244m— connection closed —\x1b[0m\r\n');
		});

		socket.addEventListener('error', () => onStatusChange?.('error'));

		const send = (message: object) => {
			if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify(message));
		};

		const onData = term.onData((data) => send({ type: 'input', data }));

		// Tell the far end the window changed, so full-screen programs redraw.
		const onResize = term.onResize(({ cols, rows }) => send({ type: 'resize', cols, rows }));

		const observer = new ResizeObserver(() => {
			try {
				fit.fit();
			} catch {
				// Fitting throws while the element is detached mid-teardown.
			}
		});
		observer.observe(host);

		return () => {
			observer.disconnect();
			onData.dispose();
			onResize.dispose();
			socket.close();
			term.dispose();
		};
	});
</script>

<div
	bind:this={host}
	class="h-[60vh] min-h-80 w-full overflow-hidden rounded-panel p-3"
	style="background-color: {THEME.background}"
></div>
