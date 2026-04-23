<script lang="ts">
	import type { SimplexLink } from '$lib/types';

	interface Props {
		links: SimplexLink[];
	}

	let { links }: Props = $props();
	let copied = $state<string | null>(null);

	async function copyLink(addr: string) {
		try {
			await navigator.clipboard.writeText(addr);
			copied = addr;
			setTimeout(() => { copied = null; }, 2000);
		} catch {
			// fallback
		}
	}
</script>

<div class="links">
	{#each links as link (link.container + link.address)}
		<button class="link-row" onclick={() => copyLink(link.address)} title="Click to copy">
			<span class="container">{link.container}</span>
			<span class="address">{link.address}</span>
			<span class="copy-hint">
				{copied === link.address ? 'copied!' : 'copy'}
			</span>
		</button>
	{/each}
</div>

<style>
	.links {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.link-row {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px 14px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius, 0);
		background: var(--color-card);
		cursor: pointer;
		transition: border-color 0.2s ease;
		width: 100%;
		text-align: left;
	}

	.link-row:hover {
		border-color: var(--color-accent);
	}

	.container {
		font-family: var(--font-mono);
		font-size: 0.7rem;
		color: var(--color-text-secondary);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		flex-shrink: 0;
	}

	.address {
		flex: 1;
		font-family: var(--font-mono);
		font-size: 0.8rem;
		color: var(--color-text);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.copy-hint {
		font-family: var(--font-mono);
		font-size: 0.65rem;
		color: var(--color-text-secondary);
		text-transform: uppercase;
		flex-shrink: 0;
	}

	.link-row:hover .copy-hint {
		color: var(--color-accent);
	}
</style>
