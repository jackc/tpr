<script>
	import { goto } from '$app/navigation';
	import { api } from '$lib/api.js';
	import { session } from '$lib/session.js';

	let username = '';
	let email = '';
	let password = '';
	let passwordConfirmation = '';

	async function register() {
		if (password !== passwordConfirmation) {
			alert('Password and confirmation must match.');
			return;
		}

		try {
			const data = await api.register({ name: username, email, password });
			session.set({ id: data.sessionID, name: data.name });
			goto('/home');
		} catch (error) {
			alert(error.data || 'Registration failed');
		}
	}
</script>

<div class="register">
	<form on:submit|preventDefault={register} autocomplete="off">
		<dl>
			<dt>
				<label for="username">User name</label>
			</dt>
			<dd>
				<input type="text" id="username" name="username" autocomplete="off" bind:value={username} />
			</dd>

			<dt>
				<label for="email">Email (optional)</label>
			</dt>
			<dd>
				<input type="email" id="email" name="email" autocomplete="off" bind:value={email} />
			</dd>

			<dt>
				<label for="password">Password</label>
			</dt>
			<dd>
				<input type="password" id="password" autocomplete="new-password" bind:value={password} />
			</dd>

			<dt>
				<label for="passwordConfirmation">Password Confirmation</label>
			</dt>
			<dd>
				<input type="password" id="passwordConfirmation" autocomplete="new-password" bind:value={passwordConfirmation} />
			</dd>
		</dl>

		<input type="submit" value="Register" />
		<a href="/login" class="login">Login</a>
	</form>
</div>
