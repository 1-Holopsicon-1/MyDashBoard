<script lang="ts">
	interface Props {
		online: boolean;
	}

	let { online }: Props = $props();
</script>

<span class="dot" class:online class:offline={!online}>
	<span class="ping"></span>
</span>

<style>
	.dot {
		position: relative;
		display: inline-block;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.online {
		background: var(--color-online, #00ff41);
		box-shadow: 0 0 6px var(--color-online, #00ff41);
	}

	.offline {
		background: var(--color-offline, #ff3333);
		box-shadow: 0 0 6px var(--color-offline, #ff3333);
	}

	.ping {
		position: absolute;
		inset: 0;
		border-radius: 50%;
		animation: pulse 2s ease-out infinite;
	}

	.online .ping {
		background: var(--color-online, #00ff41);
	}

	.offline .ping {
		background: var(--color-offline, #ff3333);
	}

	@keyframes pulse {
		0% {
			transform: scale(1);
			opacity: 0.6;
		}
		100% {
			transform: scale(2.8);
			opacity: 0;
		}
	}
</style>
