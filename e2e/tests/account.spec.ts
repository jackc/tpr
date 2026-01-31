import { test, expect } from '@playwright/test';
import { resetDatabase, createUser, queryDatabase } from '../fixtures/database';
import { login } from '../fixtures/test-helpers';

test.describe('Account Management', () => {
  test.beforeEach(async () => {
    await resetDatabase();
  });

  test('user changes password', async ({ page }) => {
    // Create user and login
    await createUser({ name: 'john', password: 'secret' });
    await login(page, 'john', 'secret');

    // Navigate to account page
    await page.getByRole('link', { name: 'Account' }).click();

    // Fill in password change form
    await page.getByLabel('Existing Password').fill('secret');
    await page.getByLabel('New Password').fill('bigsecret');
    await page.getByLabel('Password Confirmation').fill('bigsecret');

    // Handle the dialog and click update
    page.once('dialog', dialog => dialog.accept());
    await page.getByRole('button', { name: 'Update' }).click();

    // Logout
    await page.getByRole('link', { name: 'Logout' }).click();

    // Login with new password
    await login(page, 'john', 'bigsecret');

    // Verify logged in
    await expect(page.locator('p')).toContainText('No unread');
  });

  test('user fails to change password because of wrong existing password', async ({ page }) => {
    // Create user and login
    await createUser({ name: 'john', password: 'secret' });
    await login(page, 'john', 'secret');

    // Navigate to account page
    await page.getByRole('link', { name: 'Account' }).click();

    // Fill in password change form with wrong existing password
    await page.getByLabel('Existing Password').fill('wrong');
    await page.getByLabel('New Password').fill('bigsecret');
    await page.getByLabel('Password Confirmation').fill('bigsecret');

    // Handle the dialog and click update
    page.once('dialog', dialog => dialog.accept());
    await page.getByRole('button', { name: 'Update' }).click();

    // Logout
    await page.getByRole('link', { name: 'Logout' }).click();

    // Should still be able to login with OLD password (change failed)
    await login(page, 'john', 'secret');

    // Verify logged in
    await expect(page.locator('p')).toContainText('No unread');
  });

  test('user changes email', async ({ page }) => {
    // Create user and login
    await createUser({ name: 'john', password: 'secret' });
    await login(page, 'john', 'secret');

    // Navigate to account page
    await page.getByRole('link', { name: 'Account' }).click();

    // Fill in email change form (use click/type/blur for React compatibility)
    const emailField = page.locator('#email');
    await emailField.click();
    await emailField.press('Control+a');
    await emailField.type('john@example.com');
    await emailField.blur();

    await page.getByLabel('Existing Password').fill('secret');

    // Wait for the PATCH /account response
    const responsePromise = page.waitForResponse(resp =>
      resp.url().includes('/api/account') && resp.request().method() === 'PATCH'
    );

    // Handle the dialog and click update
    page.once('dialog', dialog => dialog.accept());
    await page.getByRole('button', { name: 'Update' }).click();

    // Wait for the response
    const response = await responsePromise;
    const status = response.status();

    // If request failed, get the error message
    if (status !== 200) {
      const body = await response.text();
      throw new Error(`Update failed with status ${status}: ${body}`);
    }

    // Verify email was saved in database
    const users = await queryDatabase('SELECT email FROM users WHERE name = $1', ['john']);
    expect(users[0].email).toBe('john@example.com');

    // Navigate away and back to verify email is loaded
    await page.getByRole('link', { name: 'Feeds' }).click();
    await page.getByRole('link', { name: 'Account' }).click();

    // Wait for the account page to load and populate the email field
    await expect(page.getByLabel('Existing Password')).toBeVisible();
    await page.waitForLoadState('networkidle');

    // Verify the email field is populated
    await expect(page.locator('#email')).toHaveValue('john@example.com');
  });
});
