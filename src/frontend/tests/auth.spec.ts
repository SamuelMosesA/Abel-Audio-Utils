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

test.describe('Admin Route Protection and History Navigation', () => {
  test('unauthenticated admin navigation lands on /login without rendering admin controls', async ({ page }) => {
    await page.route('**/api/auth/session', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ authenticated: false, error: 'Unauthorized session' })
        });
      } else {
        await route.continue();
      }
    });

    await page.goto('/admin');

    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByText(/Administrator Access/i)).toBeVisible();
    await expect(page.getByText(/Audio Interface Configuration/i)).not.toBeVisible();
  });

  test('authenticated session can load /admin, then logout and cannot return to it through history navigation', async ({ page }) => {
    let sessionAuthenticated = true;

    await page.route('**/api/auth/session', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        if (sessionAuthenticated) {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ authenticated: true, session_id: 'test-session-e2e', username: 'admin' })
          });
        } else {
          await route.fulfill({
            status: 401,
            contentType: 'application/json',
            body: JSON.stringify({ authenticated: false, error: 'Unauthorized session' })
          });
        }
      } else if (method === 'DELETE') {
        sessionAuthenticated = false;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'logged_out' })
        });
      } else {
        await route.continue();
      }
    });

    await page.route('**/api/audio/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
    });
    await page.route('**/api/recordings/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
    });
    await page.route('**/api/ai/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });
    await page.route('**/api/system/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ serverUrl: 'http://localhost', ssid: 'Abel' }) });
    });

    await page.goto('/admin');
    await expect(page).toHaveURL(/\/admin/);
    await expect(page.getByRole('button', { name: /Sign Out|Logout/i })).toBeVisible();

    await page.getByRole('button', { name: /Sign Out|Logout/i }).click();

    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByText(/Administrator Access/i)).toBeVisible();

    // Browser back button navigation
    await page.goBack();

    // Must be redirected back to /login and admin controls not rendered
    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByText(/Audio Interface Configuration/i)).not.toBeVisible();
  });
});

