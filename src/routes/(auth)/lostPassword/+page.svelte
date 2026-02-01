<script>
	import { api } from '$lib/api.js';

	let email = '';

	async function requestPasswordReset() {
		try {
			await api.requestPasswordReset(email);
			alert('Please check your email for reset information');
		} catch (error) {
			alert('Failure resetting password');
		}
	}
</script>

<div class="lostPassword">
	<form on:submit|preventDefault={requestPasswordReset}>
		<dl>
			<dt>
				<label for="email">Email</label>
			</dt>
			<dd>
				<input type="email" id="email" autofocus bind:value={email} />
			</dd>
		</dl>

		<input type="submit" value="Reset Password" />
	</form>
</div>
