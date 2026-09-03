<script lang="ts">
	import { theme } from '$lib/stores';
	import '$lib/themes/global.css';
	import '$lib/themes/terminal.css';
	import '$lib/themes/minimal.css';
	import '$lib/themes/glass.css';

	let { children } = $props();

	$effect(() => {
		document.documentElement.setAttribute('data-theme', $theme);
	});
</script>

<svelte:head>
	<title>holopsicon.ru</title>
</svelte:head>

{@render children()}
