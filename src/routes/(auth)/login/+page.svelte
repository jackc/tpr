<script>
	import { goto } from '$app/navigation';
	import { api } from '$lib/api.js';
	import { session } from '$lib/session.js';

	let username = '';
	let password = '';

	async function login() {
		try {
			const data = await api.login({ name: username, password });
			session.set({ id: data.sessionID, name: data.name });
			goto('/home');
		} catch (error) {
			alert(error.data || 'Login failed');
		}
	}
</script>

<div class="login">
	<form on:submit|preventDefault={login} autocomplete="off">
		<dl>
			<dt>
				<label for="username">User name</label>
			</dt>
			<dd>
				<input type="text" id="username" name="username" bind:value={username} />
			</dd>

			<dt>
				<label for="password">Password</label>
			</dt>
			<dd>
				<input type="password" id="password" name="password" bind:value={password} />
			</dd>
		</dl>

		<input type="submit" value="Login" />
		<a href="/register" class="register">Create an account</a>
		<a href="/lostPassword" class="lostPassword">Lost password</a>
	</form>
</div>
