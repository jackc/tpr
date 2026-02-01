<script>
	import { onMount } from 'svelte';
	import ArchivedItems from '$lib/models/ArchivedItems.js';
	import { toTPRString } from '$lib/utils/date.js';

	let items = [];
	let selected = null;
	let itemRefs = [];
	let collection = new ArchivedItems();
	let prevSelected = null;

	const viewItem = (item, e) => {
		if (e) {
			e.preventDefault();
		}
		window.open(item.url);
	};

	onMount(() => {
		const unsubscribe = collection.changed.subscribe(() => {
			items = collection.items;
			selected = collection.items[0] || null;
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
			}
		};

		document.addEventListener('keydown', handleKeyDown);

		return () => {
			unsubscribe();
			document.removeEventListener('keydown', handleKeyDown);
		};
	});

	// Reactive statement for selection changes
	$: {
		if (prevSelected && prevSelected !== selected) {
			prevSelected.markRead();
			ensureSelectedItemVisible();
		}
		prevSelected = selected;
	}

	function selectNext() {
		if (items.length === 0) return;
		const idx = items.indexOf(selected) + 1;
		if (idx >= items.length) return;
		selected = items[idx];
	}

	function selectPrevious() {
		if (items.length === 0) return;
		const idx = items.indexOf(selected) - 1;
		if (idx < 0) return;
		selected = items[idx];
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
</script>

<div class="home">
	{#if items.length > 0}
		<div class="pageActions">
			<div class="keyboardShortcuts">
				<dl>
					<dt>Move down:</dt>
					<dd>j</dd>
					<dt>Move up:</dt>
					<dd>k</dd>
					<dt>Open selected:</dt>
					<dd>v</dd>
				</dl>
			</div>
		</div>
	{:else}
		<div class="pageActions">
			<p class="noUnread">No archived items as of {toTPRString(new Date())}.</p>
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
			<div class="keyboardShortcuts">
				<dl>
					<dt>Move down:</dt>
					<dd>j</dd>
					<dt>Move up:</dt>
					<dd>k</dd>
					<dt>Open selected:</dt>
					<dd>v</dd>
				</dl>
			</div>
		</div>
	{/if}
</div>
