import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should show login form initially', async ({ page }) => {
    await expect(page.getByText(/Administrator Access/i)).toBeVisible();
    await expect(page.getByLabel(/Username/i)).toBeVisible();
    await expect(page.getByLabel(/Access Key/i)).toBeVisible();
  });

  test('should show error on invalid credentials', async ({ page }) => {
    await page.getByLabel(/Username/i).fill('admin');
    await page.getByLabel(/Access Key/i).fill('wrong-password');
    await page.getByRole('button', { name: /Unlock Audio Console/i }).click();

    await expect(page.getByText(/Invalid administrator credentials/i)).toBeVisible();
  });

  // Note: Successful login test would require a running backend with known credentials
  // or a mock backend. For E2E, we assume the backend might be available or we use 
  // Playwright's network mocking.
});
