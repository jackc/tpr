<script>
	import { onMount } from 'svelte';
	import { api } from '$lib/api.js';

	let email = '';
	let existingPassword = '';
	let newPassword = '';
	let passwordConfirmation = '';

	onMount(async () => {
		try {
			const data = await api.getAccount();
			email = data.email;
		} catch (error) {
			console.error('Failed to fetch account', error);
		}
	});

	async function update(e) {
		e.preventDefault();

		if (newPassword !== passwordConfirmation) {
			alert('New password and confirmation must match.');
			return;
		}

		try {
			await api.updateAccount({ email, existingPassword, newPassword });
			existingPassword = '';
			newPassword = '';
			passwordConfirmation = '';
			alert('Update succeeded');
		} catch (error) {
			alert(error.data || 'Update failed');
		}
	}
</script>

<div class="account">
	<form on:submit={update}>
		<dl>
			<dt>
				<label for="email">Email</label>
			</dt>
			<dd>
				<input type="email" name="email" id="email" bind:value={email} />
			</dd>
			<dt>
				<label for="existingPassword">Existing Password</label>
			</dt>
			<dd>
				<input type="password" name="existingPassword" id="existingPassword" bind:value={existingPassword} />
			</dd>
			<dt>
				<label for="newPassword">New Password</label>
			</dt>
			<dd>
				<input type="password" name="newPassword" id="newPassword" bind:value={newPassword} />
			</dd>
			<dt>
				<label for="passwordConfirmation">Password Confirmation</label>
			</dt>
			<dd>
				<input type="password" name="passwordConfirmation" id="passwordConfirmation" bind:value={passwordConfirmation} />
			</dd>
		</dl>

		<input type="submit" value="Update" />
	</form>
</div>
