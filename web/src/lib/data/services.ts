import { apiRequest } from '$lib/api/client';

export interface ServiceUnit {
	name: string;
	description: string;
	/** systemd's high-level state: active, inactive, failed. */
	active: string;
	/** The finer state: running, dead, exited. */
	sub: string;
	/** Whether it starts at boot: enabled, disabled, static, masked. */
	enabled: string;
	/** Units this tool refuses to stop or turn off. */
	protected: boolean;
}

export type ServiceAction = 'start' | 'stop' | 'restart' | 'enable' | 'disable';

function base(serverId: string): string {
	return `/v1/servers/${encodeURIComponent(serverId)}/services`;
}

export const servicesClient = {
	list: (serverId: string, signal?: AbortSignal) =>
		apiRequest<ServiceUnit[]>(base(serverId), { signal }),

	perform: (serverId: string, unit: string, action: ServiceAction, signal?: AbortSignal) =>
		apiRequest<void>(`${base(serverId)}/${encodeURIComponent(unit)}/actions`, {
			method: 'POST',
			body: { action },
			signal
		}),

	logs: (serverId: string, unit: string, lines = 300, signal?: AbortSignal) =>
		apiRequest<{ text: string }>(
			`${base(serverId)}/${encodeURIComponent(unit)}/logs?lines=${lines}`,
			{ signal }
		)
};
