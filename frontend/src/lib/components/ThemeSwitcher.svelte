<script lang="ts">
	import { theme } from '$lib/stores';
	import type { Theme } from '$lib/types';

	const themes: { id: Theme; label: string }[] = [
		{ id: 'terminal', label: 'Terminal' },
		{ id: 'minimal', label: 'Minimal' },
		{ id: 'glass', label: 'Glass' },
	];

	function setTheme(t: Theme) {
		theme.set(t);
	}
</script>

<div class="switcher">
	{#each themes as t}
		<button
			class="btn"
			class:active={$theme === t.id}
			onclick={() => setTheme(t.id)}
		>
			{t.label}
		</button>
	{/each}
</div>

<style>
	.switcher {
		display: flex;
		gap: 4px;
		padding: 3px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius, 0);
		background: var(--color-surface);
	}

	.btn {
		padding: 6px 14px;
		border: none;
		border-radius: var(--radius, 0);
		background: transparent;
		color: var(--color-text-secondary);
		font-family: var(--font-mono);
		font-size: 0.75rem;
		cursor: pointer;
		transition: all 0.2s ease;
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.btn:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.btn.active {
		color: var(--color-accent);
		background: var(--color-active);
		box-shadow: var(--shadow-btn, none);
	}
</style>
