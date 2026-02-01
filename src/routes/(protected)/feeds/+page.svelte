<script>
	import { onMount } from 'svelte';
	import { api } from '$lib/api.js';
	import { session } from '$lib/session.js';
	import { toTPRString } from '$lib/utils/date.js';

	let feeds = [];
	let url = '';

	onMount(() => {
		fetchFeeds();
	});

	async function fetchFeeds() {
		try {
			feeds = await api.getFeeds();
		} catch (error) {
			console.error('Failed to fetch feeds', error);
		}
	}

	async function subscribe(e) {
		e.preventDefault();

		try {
			await api.subscribe(url);
			url = '';
			await fetchFeeds();
		} catch (error) {
			alert('Subscription failed');
		}
	}

	async function unsubscribe(feed, e) {
		e.preventDefault();
		if (confirm('Are you sure you want to unsubscribe from ' + feed.name + '?')) {
			await api.deleteSubscription(feed.feed_id);
			feeds = feeds.filter((f) => f !== feed);
		}
	}

	async function importOPML(e) {
		e.preventDefault();
		const formData = new FormData(e.target);

		try {
			await api.importOPML(formData);
			await fetchFeeds();
			alert('import success');
		} catch (error) {
			alert('Import failed');
		}
	}
</script>

<div class="feeds">
	<form class="subscribe" on:submit={subscribe}>
		<dl>
			<dt>
				<label for="feed_url">Feed URL</label>
			</dt>
			<dd>
				<input type="text" id="feed_url" bind:value={url} />
			</dd>
		</dl>
		<input type="submit" value="Subscribe" />
	</form>

	<form class="import" on:submit={importOPML}>
		<dl>
			<dt>
				<label for="opml_file">OPML File</label>
			</dt>
			<dd><input type="file" name="file" id="opml_file" /></dd>
		</dl>
		<input type="submit" value="Import" />
		{' '}
		<a href="/api/feeds.xml?session={$session.id}">Export</a>
	</form>

	<ul>
		{#each feeds as feed (feed.url)}
			<li>
				<div class="name"><a href={feed.url}>{feed.name}</a></div>
				{#if feed.last_publication_time}
					<div class="meta">
						Last published
						{' '}
						<time datetime={feed.last_publication_time.toISOString()}>
							{toTPRString(feed.last_publication_time)}
						</time>
					</div>
				{/if}
				{#if feed.failure_count > 0}
					<div class="error">{feed.last_failure}</div>
				{/if}
				<div class="actions">
					<a href="#" on:click={(e) => unsubscribe(feed, e)}>Unsubscribe</a>
				</div>
			</li>
		{/each}
	</ul>
</div>
