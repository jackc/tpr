<script>
	import { ajaxPending } from '$lib/api.js';
</script>

<div id="working_notice" style="display: {$ajaxPending > 0 ? '' : 'none'}">
	<div>Working...</div>
</div>
