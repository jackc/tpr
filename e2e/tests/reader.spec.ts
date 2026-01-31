import { test, expect } from '@playwright/test';
import { resetDatabase, createUser, createFeed, createItem, queryDatabase } from '../fixtures/database';
import { login } from '../fixtures/test-helpers';

test.describe('Reader Functionality', () => {
  test.beforeEach(async () => {
    await resetDatabase();
  });

  test('user marks all items read', async ({ page }) => {
    // Create user, feed, and subscription
    const user = await createUser({ name: 'john', password: 'secret' });
    const feed = await createFeed({});
    await queryDatabase(
      'INSERT INTO subscriptions (user_id, feed_id) VALUES ($1, $2)',
      [user.id, feed.id]
    );

    // Create first item and mark as unread
    const beforeItem = await createItem({
      feed_id: feed.id,
      title: 'First Post',
      publication_time: '2014-02-06T10:34:51',
    });
    await queryDatabase(
      'INSERT INTO unread_items (user_id, feed_id, item_id) VALUES ($1, $2, $3)',
      [user.id, feed.id, beforeItem.id]
    );

    // Login and verify first item is shown
    await login(page, 'john', 'secret');
    await expect(page.locator('body')).toContainText('First Post');
    await expect(page.locator('body')).toContainText('February 6th, 2014 at 10:34 am');

    // Add a second item while user is viewing
    const afterItem = await createItem({
      feed_id: feed.id,
      title: 'Second Post',
    });
    await queryDatabase(
      'INSERT INTO unread_items (user_id, feed_id, item_id) VALUES ($1, $2, $3)',
      [user.id, feed.id, afterItem.id]
    );

    // Mark all read - should show second item
    await page.getByRole('link', { name: 'Mark All Read' }).click();
    await expect(page.locator('body')).toContainText('Second Post');

    // Mark all read again - should show no items
    await page.getByRole('link', { name: 'Mark All Read' }).click();
    await expect(page.locator('body')).toContainText('Refresh');

    // Add a third item
    const anotherItem = await createItem({
      feed_id: feed.id,
      title: 'Third Post',
    });
    await queryDatabase(
      'INSERT INTO unread_items (user_id, feed_id, item_id) VALUES ($1, $2, $3)',
      [user.id, feed.id, anotherItem.id]
    );

    // Refresh and verify third item appears
    await page.getByRole('link', { name: 'Refresh' }).click();
    await expect(page.locator('body')).toContainText('Third Post');
  });

  test('user uses keyboard shortcuts', async ({ page }) => {
    // Create user, feed, and subscription
    const user = await createUser({ name: 'john', password: 'secret' });
    const feed = await createFeed({});
    await queryDatabase(
      'INSERT INTO subscriptions (user_id, feed_id) VALUES ($1, $2)',
      [user.id, feed.id]
    );

    // Create two items
    const firstItem = await createItem({
      feed_id: feed.id,
      title: 'First Post',
      publication_time: new Date(Date.now() - 5000).toISOString(),
    });
    await queryDatabase(
      'INSERT INTO unread_items (user_id, feed_id, item_id) VALUES ($1, $2, $3)',
      [user.id, feed.id, firstItem.id]
    );

    const secondItem = await createItem({
      feed_id: feed.id,
      title: 'Second Post',
      publication_time: new Date().toISOString(),
    });
    await queryDatabase(
      'INSERT INTO unread_items (user_id, feed_id, item_id) VALUES ($1, $2, $3)',
      [user.id, feed.id, secondItem.id]
    );

    // Login and verify first item is selected
    await login(page, 'john', 'secret');
    await expect(page.locator('.selected')).toContainText('First Post');
    await expect(page.locator('.selected')).not.toContainText('Second Post');

    // Press 'j' to move to next item
    await page.keyboard.press('j');

    // Verify second item is now selected
    await expect(page.locator('.selected')).not.toContainText('First Post');
    await expect(page.locator('.selected')).toContainText('Second Post');

    // Navigate away and back
    await page.getByRole('link', { name: 'Feeds' }).click();
    await page.getByRole('link', { name: 'Home' }).click();

    // First post should be gone (was auto-marked as read), second should still be there
    await expect(page.locator('body')).not.toContainText('First Post');
    await expect(page.locator('body')).toContainText('Second Post');

    // Press Shift+A to mark all read
    await page.keyboard.press('Shift+A');

    // Second post should be gone
    await expect(page.locator('body')).not.toContainText('Second Post');
  });

  test('user looks at archived posts', async ({ page }) => {
    // Create user, feed, and subscription
    const user = await createUser({ name: 'john', password: 'secret' });
    const feed = await createFeed({});
    await queryDatabase(
      'INSERT INTO subscriptions (user_id, feed_id) VALUES ($1, $2)',
      [user.id, feed.id]
    );

    // Create item and mark as unread
    const item = await createItem({
      feed_id: feed.id,
      title: 'First Post',
      publication_time: '2014-02-06T10:34:51',
    });
    await queryDatabase(
      'INSERT INTO unread_items (user_id, feed_id, item_id) VALUES ($1, $2, $3)',
      [user.id, feed.id, item.id]
    );

    // Login and verify item is shown
    await login(page, 'john', 'secret');
    await expect(page.locator('body')).toContainText('First Post');
    await expect(page.locator('body')).toContainText('February 6th, 2014 at 10:34 am');

    // Mark all read
    await page.getByRole('link', { name: 'Mark All Read' }).click();
    await expect(page.locator('body')).toContainText('Refresh');

    // Go to Archive
    await page.getByRole('link', { name: 'Archive' }).click();

    // Verify archived item is shown
    await expect(page.locator('body')).toContainText('First Post');
  });
});
