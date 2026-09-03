<script lang="ts">
	import type { ContainerStatus } from '$lib/types';
	import StatusDot from './StatusDot.svelte';

	interface Props {
		container: ContainerStatus;
	}

	let { container }: Props = $props();

	const displayName = $derived(container.name.replace(/[-_]/g, ' '));
</script>

<div class="card" class:online={container.online} class:offline={!container.online}>
	<div class="header">
		<StatusDot online={container.online} />
		<span class="name">{displayName}</span>
	</div>

	<div class="details">
		<div class="row">
			<span class="label">State</span>
			<span class="value state" class:up={container.online} class:down={!container.online}>
				{container.state}
			</span>
		</div>
		<div class="row">
			<span class="label">Image</span>
			<span class="value mono image">{container.image}</span>
		</div>
		<div class="row">
			<span class="label">Uptime</span>
			<span class="value">{container.status}</span>
		</div>
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
		gap: 8px;
	}

	.label {
		color: var(--color-text-secondary);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		font-size: 0.7rem;
		font-family: var(--font-mono);
		flex-shrink: 0;
	}

	.value {
		color: var(--color-text);
		font-size: 0.8rem;
		text-align: right;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.mono {
		font-family: var(--font-mono);
	}

	.image {
		font-size: 0.7rem;
		max-width: 160px;
	}

	.state {
		font-family: var(--font-mono);
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.up {
		color: var(--color-online);
	}

	.down {
		color: var(--color-offline);
	}
</style>
