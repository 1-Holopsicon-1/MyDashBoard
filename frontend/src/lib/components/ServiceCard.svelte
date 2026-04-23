<script lang="ts">
	import type { ServiceStatus } from '$lib/types';
	import StatusDot from './StatusDot.svelte';

	interface Props {
		service: ServiceStatus;
	}

	let { service }: Props = $props();

	const latencyClass = $derived(
		service.latency_ms < 100 ? 'fast' : service.latency_ms < 500 ? 'medium' : 'slow'
	);
</script>

<div class="card" class:online={service.online} class:offline={!service.online}>
	<div class="header">
		<StatusDot online={service.online} />
		<span class="name">{service.name}</span>
	</div>

	<div class="details">
		<div class="row">
			<span class="label">Status</span>
			<span class="value status-text" class:up={service.online} class:down={!service.online}>
				{service.online ? 'UP' : 'DOWN'}
			</span>
		</div>

		{#if service.online}
			<div class="row">
				<span class="label">Latency</span>
				<span class="value latency {latencyClass}">{service.latency_ms}ms</span>
			</div>
			<div class="row">
				<span class="label">HTTP</span>
				<span class="value mono">{service.code}</span>
			</div>
		{/if}
	</div>
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

	.name {
		font-family: var(--font-mono);
		font-size: 0.95rem;
		font-weight: 600;
		color: var(--color-text);
		text-transform: capitalize;
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

	.status-text {
		font-family: var(--font-mono);
		font-weight: 700;
		letter-spacing: 0.08em;
	}

	.up {
		color: var(--color-online);
	}

	.down {
		color: var(--color-offline);
	}

	.latency {
		font-family: var(--font-mono);
	}

	.latency.fast {
		color: var(--color-online);
	}

	.latency.medium {
		color: var(--color-warning, #ffaa00);
	}

	.latency.slow {
		color: var(--color-offline);
	}
</style>
