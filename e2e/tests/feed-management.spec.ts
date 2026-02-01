import { test, expect } from '@playwright/test';
import { resetDatabase, createUser } from '../fixtures/database';
import { login } from '../fixtures/test-helpers';
import path from 'path';

test.describe('Feed Management', () => {
  test.beforeEach(async () => {
    await resetDatabase();
  });

  test('user subscribes to a feed', async ({ page }) => {
    // Create user and login
    await createUser({ name: 'john', password: 'secret' });
    await login(page, 'john', 'secret');

    // Navigate to feeds page
    await page.getByRole('link', { name: 'Feeds' }).click();

    // Wait for feeds page to load
    await expect(page.getByLabel('Feed URL')).toBeVisible();

    // Subscribe to a feed
    await page.getByLabel('Feed URL').fill('http://localhost:1234');
    await page.getByRole('button', { name: 'Subscribe' }).click();

    // Verify feed appears in the list
    await expect(page.locator('.feeds > ul')).toContainText('http://localhost:1234');
  });

  test('user imports OPML file', async ({ page }) => {
    // Create user and login
    await createUser({ name: 'john', password: 'secret' });
    await login(page, 'john', 'secret');

    // Navigate to feeds page
    await page.getByRole('link', { name: 'Feeds' }).click();

    // Wait for feeds page to load
    await expect(page.locator('[type=file]')).toBeVisible();

    // Upload OPML file
    const opmlPath = path.join(__dirname, '../../test/testdata/opml.xml');
    await page.locator('[type=file]').setInputFiles(opmlPath);

    // Handle dialog and click import
    page.once('dialog', dialog => dialog.accept());
    await page.getByRole('button', { name: 'Import' }).click();

    // Verify feeds from OPML appear in the list
    await expect(page.locator('.feeds > ul')).toContainText('http://localhost/rss');
    await expect(page.locator('.feeds > ul')).toContainText('http://localhost/other/rss');
  });
});
