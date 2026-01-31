import { test, expect } from '@playwright/test';
import { resetDatabase, createUser, queryDatabase } from '../fixtures/database';
import { login } from '../fixtures/test-helpers';

test.describe('Session Management', () => {
  test.beforeEach(async () => {
    await resetDatabase();
  });

  test('user with invalid session is logged out', async ({ page }) => {
    // Create user and login
    await createUser({ name: 'john', password: 'secret' });
    await login(page, 'john', 'secret');

    // Verify logged in (should see Refresh button)
    await expect(page.locator('body')).toContainText('Refresh');

    // Delete the session from database
    await queryDatabase('DELETE FROM sessions');

    // Try to navigate - should be logged out
    await page.getByRole('link', { name: 'Feeds' }).click();

    // Verify redirected to login page
    await expect(page.getByLabel('User name')).toBeVisible();
    await expect(page.getByLabel('Password')).toBeVisible();
  });
});
