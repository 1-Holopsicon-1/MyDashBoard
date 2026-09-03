<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { fetchDevices, fetchServices, fetchContainers, fetchSimplexLinks, fetchAuthStatus } from '$lib/api';
	import {
		devices,
		services,
		containers,
		simplexLinks,
		devicesLoading,
		servicesLoading,
		containersLoading,
		simplexLoading,
		devicesError,
		servicesError,
		containersError,
		simplexError,
		lastUpdated,
		authStatus,
	} from '$lib/stores';
	import ThemeSwitcher from '$lib/components/ThemeSwitcher.svelte';
	import LoginCard from '$lib/components/LoginCard.svelte';
	import DeviceCard from '$lib/components/DeviceCard.svelte';
	import ServiceCard from '$lib/components/ServiceCard.svelte';
	import ContainerCard from '$lib/components/ContainerCard.svelte';
	import SimplexLinks from '$lib/components/SimplexLinks.svelte';

	let showAuth = $state(false);
	let clickTimes: number[] = [];

	function handleLogoClick() {
		if (showAuth) return;
		const now = Date.now();
		clickTimes = clickTimes.filter(t => now - t < 10_000);
		clickTimes.push(now);
		if (clickTimes.length >= 5) {
			showAuth = true;
			clickTimes = [];
		}
	}

	function handleLogoKey(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			handleLogoClick();
		}
	}

	let devicesTimer: ReturnType<typeof setInterval>;
	let servicesTimer: ReturnType<typeof setInterval>;
	let containersTimer: ReturnType<typeof setInterval>;
	let simplexTimer: ReturnType<typeof setInterval>;

	let loadDevicesSeq = 0;
	let loadServicesSeq = 0;
	let loadContainersSeq = 0;
	let loadSimplexSeq = 0;
	let abortController = new AbortController();
	let authKnown = $state(false);

	const showIP = $derived($authStatus.authenticated);

	$effect(() => {
		if (authKnown && $authStatus.authenticated) {
			loadDevices();
		}
	});

	async function loadAuth() {
		try {
			const status = await fetchAuthStatus(abortController.signal);
			authStatus.set(status);
		} catch { /* ignore */ }
		authKnown = true;
	}

	async function loadDevices() {
		const seq = ++loadDevicesSeq;
		devicesError.set(null);
		try {
			const data = await fetchDevices(abortController.signal);
			if (seq !== loadDevicesSeq) return;
			devices.set(data);
			lastUpdated.set(new Date());
		} catch (e) {
			if (seq !== loadDevicesSeq) return;
			if (e instanceof DOMException && e.name === 'AbortError') return;
			devicesError.set(e instanceof Error ? e.message : 'Failed to load devices');
		} finally {
			if (seq === loadDevicesSeq) devicesLoading.set(false);
		}
	}

	async function loadServices() {
		const seq = ++loadServicesSeq;
		servicesError.set(null);
		try {
			const data = await fetchServices(abortController.signal);
			if (seq !== loadServicesSeq) return;
			services.set(data);
			lastUpdated.set(new Date());
		} catch (e) {
			if (seq !== loadServicesSeq) return;
			if (e instanceof DOMException && e.name === 'AbortError') return;
			servicesError.set(e instanceof Error ? e.message : 'Failed to load services');
		} finally {
			if (seq === loadServicesSeq) servicesLoading.set(false);
		}
	}

	async function loadContainers() {
		const seq = ++loadContainersSeq;
		containersError.set(null);
		try {
			const data = await fetchContainers(abortController.signal);
			if (seq !== loadContainersSeq) return;
			containers.set(data);
			lastUpdated.set(new Date());
		} catch (e) {
			if (seq !== loadContainersSeq) return;
			if (e instanceof DOMException && e.name === 'AbortError') return;
			containersError.set(e instanceof Error ? e.message : 'Failed to load containers');
		} finally {
			if (seq === loadContainersSeq) containersLoading.set(false);
		}
	}

	async function loadSimplex() {
		const seq = ++loadSimplexSeq;
		simplexError.set(null);
		try {
			const data = await fetchSimplexLinks(abortController.signal);
			if (seq !== loadSimplexSeq) return;
			simplexLinks.set(data);
			lastUpdated.set(new Date());
		} catch (e) {
			if (seq !== loadSimplexSeq) return;
			if (e instanceof DOMException && e.name === 'AbortError') return;
			simplexError.set(e instanceof Error ? e.message : 'Failed to load SimpleX links');
		} finally {
			if (seq === loadSimplexSeq) simplexLoading.set(false);
		}
	}

	onMount(async () => {
		await loadAuth();
		loadDevices();
		loadServices();
		loadContainers();
		loadSimplex();
		devicesTimer = setInterval(loadDevices, 30_000);
		servicesTimer = setInterval(loadServices, 60_000);
		containersTimer = setInterval(loadContainers, 30_000);
		simplexTimer = setInterval(loadSimplex, 60_000);
	});

	onDestroy(() => {
		abortController.abort();
		clearInterval(devicesTimer);
		clearInterval(servicesTimer);
		clearInterval(containersTimer);
		clearInterval(simplexTimer);
	});

	function formatTime(date: Date | null): string {
		if (!date) return '—';
		return date.toLocaleTimeString('ru-RU', {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
		});
	}
</script>

<div class="layout">
	<header>
		<span class="logo" role="button" tabindex="0" onclick={handleLogoClick} onkeydown={handleLogoKey}>holopsicon.ru</span>
		<div class="header-right">
			<LoginCard {showAuth} />
			<ThemeSwitcher />
		</div>
	</header>

	<main>
		<!-- Network -->
		<section>
			<div class="section-header">
				<h2 class="section-title">Network</h2>
				{#if $devicesLoading}
					<span class="badge loading-badge">loading…</span>
				{:else}
					<span class="badge count-badge">{$devices.length} devices</span>
				{/if}
			</div>

			{#if $devicesError}
				<div class="error-block">
					<span class="error-icon">!</span>
					<span>{$devicesError}</span>
				</div>
			{:else if $devicesLoading}
				<div class="grid">
					{#each Array(4) as _}
						<div class="skeleton card"></div>
					{/each}
				</div>
			{:else}
				<div class="grid">
					{#each $devices as device (device.id)}
						<DeviceCard {device} showIP={showIP} />
					{/each}
				</div>
			{/if}
		</section>

		<!-- Containers -->
		<section>
			<div class="section-header">
				<h2 class="section-title">Containers</h2>
				{#if $containersLoading}
					<span class="badge loading-badge">loading…</span>
				{:else}
					<span class="badge count-badge">{$containers.length} containers</span>
				{/if}
			</div>

			{#if $containersError}
				<div class="error-block">
					<span class="error-icon">!</span>
					<span>{$containersError}</span>
				</div>
			{:else if $containersLoading}
				<div class="grid">
					{#each Array(2) as _}
						<div class="skeleton card"></div>
					{/each}
				</div>
			{:else}
				<div class="grid">
					{#each $containers as container (container.name)}
						<ContainerCard {container} />
					{/each}
				</div>
			{/if}
		</section>

		<!-- SimpleX Links -->
		{#if $authStatus.authenticated && $simplexLinks.length > 0}
			<section>
				<div class="section-header">
					<h2 class="section-title">SimpleX</h2>
					<span class="badge count-badge">{$simplexLinks.length} servers</span>
				</div>

				{#if $simplexError}
					<div class="error-block">
						<span class="error-icon">!</span>
						<span>{$simplexError}</span>
					</div>
				{:else}
					<SimplexLinks links={$simplexLinks} />
				{/if}
			</section>
		{/if}

		<!-- Services -->
		<section>
			<div class="section-header">
				<h2 class="section-title">Services</h2>
				{#if $servicesLoading}
					<span class="badge loading-badge">loading…</span>
				{:else}
					<span class="badge count-badge">{$services.length} services</span>
				{/if}
			</div>

			{#if $servicesError}
				<div class="error-block">
					<span class="error-icon">!</span>
					<span>{$servicesError}</span>
				</div>
			{:else if $servicesLoading}
				<div class="grid">
					{#each Array(2) as _}
						<div class="skeleton card"></div>
					{/each}
				</div>
			{:else}
				<div class="grid">
					{#each $services as service (service.name)}
						<ServiceCard {service} />
					{/each}
				</div>
			{/if}
		</section>
	</main>

	<footer>
		<span>Last update: {formatTime($lastUpdated)}</span>
	</footer>
</div>

<style>
	.layout {
		display: flex;
		flex-direction: column;
		min-height: 100vh;
		padding: 0 24px;
		max-width: 1200px;
		margin: 0 auto;
	}

	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 20px 0;
		margin-bottom: 8px;
	}

	.header-right {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.logo {
		font-family: var(--font-mono);
		font-size: 1.1rem;
		font-weight: 700;
		color: var(--color-text);
		letter-spacing: -0.02em;
	}

	main {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 40px;
	}

	section {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.section-title {
		font-family: var(--font-mono);
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--color-text-secondary);
		text-transform: uppercase;
		letter-spacing: 0.1em;
	}

	.badge {
		font-family: var(--font-mono);
		font-size: 0.65rem;
		padding: 2px 8px;
		border-radius: var(--radius, 0);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.count-badge {
		color: var(--color-text-secondary);
		border: 1px solid var(--color-border);
	}

	.loading-badge {
		color: var(--color-accent);
		border: 1px solid var(--color-accent);
		opacity: 0.7;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		gap: 12px;
	}

	.skeleton {
		min-height: 130px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius, 0);
		animation: shimmer 1.5s ease-in-out infinite alternate;
	}

	@keyframes shimmer {
		from { opacity: 0.5; }
		to { opacity: 1; }
	}

	.error-block {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 14px 16px;
		border: 1px solid var(--color-offline);
		border-radius: var(--radius, 0);
		background: var(--color-surface);
		color: var(--color-offline);
		font-family: var(--font-mono);
		font-size: 0.8rem;
	}

	.error-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		border: 1px solid var(--color-offline);
		font-weight: 700;
		font-size: 0.75rem;
		flex-shrink: 0;
	}

	footer {
		padding: 24px 0;
		margin-top: 40px;
		text-align: center;
		font-family: var(--font-mono);
		font-size: 0.7rem;
		color: var(--color-text-secondary);
		letter-spacing: 0.03em;
	}

	@media (max-width: 600px) {
		.layout {
			padding: 0 16px;
		}

		header {
			flex-direction: column;
			gap: 12px;
			align-items: flex-start;
		}

		.header-right {
			width: 100%;
			justify-content: space-between;
		}

		.grid {
			grid-template-columns: 1fr;
		}
	}
</style>
