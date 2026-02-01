<script>
	import Header from '$lib/components/Header.svelte';
	import WorkingNotice from '$lib/components/WorkingNotice.svelte';
	import '../styles/main.scss';
</script>

<div>
	<WorkingNotice />
	<Header />
	<slot />
</div>

<footer>
	<a href="https://github.com/jackc/tpr">The Pithy Reader</a>
	© 2013-2026
	<a href="http://www.jackchristensen.com/">Jack Christensen</a>
</footer>
