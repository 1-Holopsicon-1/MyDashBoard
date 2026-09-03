<script lang="ts">
	import type { TailscaleDevice } from '$lib/types';
	import StatusDot from './StatusDot.svelte';

	interface Props {
		device: TailscaleDevice;
		showIP?: boolean;
	}

	let { device, showIP = false }: Props = $props();

	const isOnline = $derived(device.connectedToControl);
	const displayName = $derived(device.hostname || device.name.split('.')[0]);
	const primaryIP = $derived(device.addresses?.[0] ?? '—');

	function formatLastSeen(iso: string): string {
		const date = new Date(iso);
		const now = Date.now();
		const diffMs = now - date.getTime();
		const diffMin = Math.floor(diffMs / 60000);

		if (diffMin < 1) return 'just now';
		if (diffMin < 60) return `${diffMin}m ago`;
		const diffH = Math.floor(diffMin / 60);
		if (diffH < 24) return `${diffH}h ago`;
		const diffD = Math.floor(diffH / 24);
		return `${diffD}d ago`;
	}
</script>

<div class="card" class:online={isOnline} class:offline={!isOnline}>
	<div class="header">
		<StatusDot online={isOnline} />
		<span class="hostname">{displayName}</span>
	</div>

	<div class="details">
		<div class="row">
			<span class="label">OS</span>
			<span class="value">{device.os}</span>
		</div>
		{#if showIP && device.addresses?.length}
			<div class="row">
				<span class="label">IP</span>
				<span class="value mono">{primaryIP}</span>
			</div>
		{/if}
		<div class="row">
			<span class="label">Seen</span>
			<span class="value">{formatLastSeen(device.lastSeen)}</span>
		</div>
	</div>

	{#if !device.authorized}
		<div class="badge unauthorized">unauthorized</div>
	{/if}
</div>

<style>
	.card {
		padding: 16px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius, 0);
		background: var(--color-card);
		transition: border-color 0.2s ease, box-shadow 0.2s ease;
	}

	.card:hover {
		border-color: var(--color-accent);
		box-shadow: var(--shadow-card-hover, none);
	}

	.header {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-bottom: 14px;
	}

	.hostname {
		font-family: var(--font-mono);
		font-size: 0.95rem;
		font-weight: 600;
		color: var(--color-text);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.details {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.8rem;
	}

	.label {
		color: var(--color-text-secondary);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		font-size: 0.7rem;
		font-family: var(--font-mono);
	}

	.value {
		color: var(--color-text);
		font-size: 0.8rem;
	}

	.mono {
		font-family: var(--font-mono);
	}

	.badge {
		margin-top: 10px;
		padding: 2px 8px;
		font-size: 0.65rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		font-family: var(--font-mono);
		border-radius: var(--radius, 0);
		display: inline-block;
	}

	.unauthorized {
		color: var(--color-offline);
		border: 1px solid var(--color-offline);
		opacity: 0.8;
	}
</style>
