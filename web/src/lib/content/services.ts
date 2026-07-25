import type { LucideProps } from '@lucide/svelte';
import ChartColumn from '@lucide/svelte/icons/chart-column';
import ChartLine from '@lucide/svelte/icons/chart-line';
import Database from '@lucide/svelte/icons/database';
import Gauge from '@lucide/svelte/icons/gauge';
import HardDrive from '@lucide/svelte/icons/hard-drive';
import KeyRound from '@lucide/svelte/icons/key-round';
import Layers from '@lucide/svelte/icons/layers';
import Radio from '@lucide/svelte/icons/radio';
import Route from '@lucide/svelte/icons/route';
import Search from '@lucide/svelte/icons/search';
import Sparkles from '@lucide/svelte/icons/sparkles';
import Video from '@lucide/svelte/icons/video';
import type { Component } from 'svelte';

export interface ServiceHighlight {
	name: string;
	/** What it does, in plain words — not which category it belongs to. */
	purpose: string;
	icon: Component<LucideProps>;
}

/** A slice of the one-click catalogue, shown on the landing page. */
export const serviceHighlights: ServiceHighlight[] = [
	{ name: 'PostgreSQL', purpose: 'Stores your data', icon: Database },
	{ name: 'Redis', purpose: 'Makes things fast', icon: Gauge },
	{ name: 'ClickHouse', purpose: 'Crunches big numbers', icon: ChartColumn },
	{ name: 'MediaMTX', purpose: 'Streams live video', icon: Video },
	{ name: 'Traefik', purpose: 'Directs web traffic', icon: Route },
	{ name: 'MinIO', purpose: 'Holds files and uploads', icon: HardDrive },
	{ name: 'NATS', purpose: 'Passes messages around', icon: Radio },
	{ name: 'Meilisearch', purpose: 'Powers your search box', icon: Search },
	{ name: 'Grafana', purpose: 'Draws the charts', icon: ChartLine },
	{ name: 'Keycloak', purpose: 'Handles sign-ins', icon: KeyRound },
	{ name: 'RabbitMQ', purpose: 'Queues background work', icon: Layers },
	{ name: 'Qdrant', purpose: 'Search that understands meaning', icon: Sparkles }
];
