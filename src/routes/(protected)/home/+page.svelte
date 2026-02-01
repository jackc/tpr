<script>
	import { onMount } from 'svelte';
	import UnreadItems from '$lib/models/UnreadItems.js';
	import { toTPRString } from '$lib/utils/date.js';

	let items = [];
	let selected = null;
	let itemRefs = [];
	let collection = new UnreadItems();
	let prevSelected = null;

	const viewItem = (item, e) => {
		if (e) {
			e.preventDefault();
		}
		item.markRead();
		window.open(item.url);
	};

	onMount(() => {
		const unsubscribe = collection.changed.subscribe(() => {
			items = collection.items;
			const newSelected = collection.items[0] || null;
			selected = newSelected;
			// Set prevSelected to track the initial selection
			prevSelected = newSelected;
		});

		collection.fetch();

		const handleKeyDown = (e) => {
			switch (e.which) {
				case 74: // j
					selectNext();
					break;
				case 75: // k
					selectPrevious();
					break;
				case 86: // v
					viewSelected();
					break;
				case 65: // a
					if (e.shiftKey) {
						collection.markAllRead();
					}
					break;
			}
		};

		document.addEventListener('keydown', handleKeyDown);

		return () => {
			unsubscribe();
			document.removeEventListener('keydown', handleKeyDown);
		};
	});

	// Track selection changes and mark previous as read
	function updateSelection(newSelected) {
		if (prevSelected && prevSelected !== newSelected) {
			prevSelected.markRead();
		}
		prevSelected = newSelected;
		selected = newSelected;
		ensureSelectedItemVisible();
	}

	function selectNext() {
		if (items.length === 0) return;
		const idx = items.indexOf(selected) + 1;
		if (idx >= items.length) return;
		updateSelection(items[idx]);
	}

	function selectPrevious() {
		if (items.length === 0) return;
		const idx = items.indexOf(selected) - 1;
		if (idx < 0) return;
		updateSelection(items[idx]);
	}

	function viewSelected() {
		if (selected) {
			viewItem(selected);
		}
	}

	function ensureSelectedItemVisible() {
		if (!selected) return;
		const idx = items.indexOf(selected);
		const el = itemRefs[idx];
		if (!el) return;

		const rect = el.getBoundingClientRect();
		const entirelyVisible =
			rect.top >= 0 &&
			rect.left >= 0 &&
			rect.bottom <= window.innerHeight &&
			rect.right <= window.innerWidth;

		if (!entirelyVisible) {
			el.scrollIntoView();
		}
	}

	function markAllRead(e) {
		e.preventDefault();
		collection.markAllRead();
	}

	function refresh(e) {
		e.preventDefault();
		collection.fetch();
	}
</script>

<div class="home">
	{#if items.length > 0}
		<div class="pageActions">
			<a href="#" class="markAllRead button" on:click={markAllRead}> Mark All Read </a>
			<div class="keyboardShortcuts">
				<dl>
					<dt>Move down:</dt>
					<dd>j</dd>
					<dt>Move up:</dt>
					<dd>k</dd>
					<dt>Open selected:</dt>
					<dd>v</dd>
					<dt>Mark all read:</dt>
					<dd>shift+a</dd>
				</dl>
			</div>
		</div>
	{:else}
		<div class="pageActions">
			<a href="#" class="refresh button" on:click={refresh}>Refresh</a>
			<p class="noUnread">No unread items as of {toTPRString(new Date())}.</p>
		</div>
	{/if}

	<ul class="unreadItems">
		{#each items as item, index (item.id)}
			<li bind:this={itemRefs[index]} class:selected={item === selected}>
				<div class="title">
					<a href={item.url} on:click={(e) => viewItem(item, e)}>{item.title}</a>
				</div>
				<span class="meta">
					<span class="feedName">{item.feed_name}</span>
					{' '}
					on
					{' '}
					<time datetime={item.publication_time.toISOString()} class="publication">
						{toTPRString(item.publication_time)}
					</time>
				</span>
			</li>
		{/each}
	</ul>

	{#if items.length > 15}
		<div class="pageActions">
			<a href="#" class="markAllRead button" on:click={markAllRead}> Mark All Read </a>
			<div class="keyboardShortcuts">
				<dl>
					<dt>Move down:</dt>
					<dd>j</dd>
					<dt>Move up:</dt>
					<dd>k</dd>
					<dt>Open selected:</dt>
					<dd>v</dd>
					<dt>Mark all read:</dt>
					<dd>shift+a</dd>
				</dl>
			</div>
		</div>
	{/if}
</div>
