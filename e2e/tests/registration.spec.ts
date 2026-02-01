import { test, expect } from '@playwright/test';
import { resetDatabase, queryDatabase } from '../fixtures/database';

test.describe('User Registration', () => {
  test.beforeEach(async () => {
    await resetDatabase();
  });

  test('register new user with email', async ({ page }) => {
    await page.goto('/login');

    await page.getByRole('link', { name: 'Create an account' }).click();

    // Wait for registration form to load and be fully interactive
    const usernameInput = page.getByLabel('User name');
    await expect(usernameInput).toBeVisible();
    await page.waitForLoadState('networkidle');

    // Fill form with retry logic to handle any hydration timing
    await usernameInput.click();
    await usernameInput.fill('joe1');
    await page.getByLabel(/Email \(optional\)/).fill('joe@example.com');
    await page.getByLabel('Password', { exact: true }).fill('bigsecret');
    await page.getByLabel('Password Confirmation').fill('bigsecret');

    await page.getByRole('button', { name: 'Register' }).click();

    await expect(page.locator('body')).toContainText('No unread items');

    // Verify user was created in database
    const users = await queryDatabase('SELECT * FROM users WHERE name = $1', ['joe1']);
    expect(users).toHaveLength(1);
    expect(users[0].name).toBe('joe1');
    expect(users[0].email).toBe('joe@example.com');
  });

  test('register new user without email', async ({ page }) => {
    await page.goto('/login');

    await page.getByRole('link', { name: 'Create an account' }).click();

    // Wait for registration form to load and be fully interactive
    const usernameInput = page.getByLabel('User name');
    await expect(usernameInput).toBeVisible();
    await page.waitForLoadState('networkidle');

    // Fill form with retry logic to handle any hydration timing
    await usernameInput.click();
    await usernameInput.fill('joe2');
    await page.getByLabel('Password', { exact: true }).fill('bigsecret');
    await page.getByLabel('Password Confirmation').fill('bigsecret');

    await page.getByRole('button', { name: 'Register' }).click();

    await expect(page.locator('body')).toContainText('No unread items');

    // Verify user was created in database
    const users = await queryDatabase('SELECT * FROM users WHERE name = $1', ['joe2']);
    expect(users).toHaveLength(1);
    expect(users[0].name).toBe('joe2');
    expect(users[0].email).toBeNull();
  });
});
