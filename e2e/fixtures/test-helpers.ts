import { Page, expect } from '@playwright/test';

/**
 * Log in a user via the login form
 * @param page Playwright page object
 * @param username Username to log in with
 * @param password Password to log in with
 */
export async function login(
  page: Page,
  username: string,
  password: string
): Promise<void> {
  await page.goto('/#login');
  await page.getByLabel('User name').fill(username);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Login' }).click();

  // Wait for login to complete by checking for element that always appears after login
  await expect(page.getByRole('link', { name: 'Logout' })).toBeVisible();
}
