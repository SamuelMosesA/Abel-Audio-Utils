import { expect, test } from '@playwright/test';

test.describe('Safari-safe live audio controls', () => {
  test('uses HLS and starts only after an explicit user action', async ({ page }) => {
    await page.addInitScript(() => {
      HTMLMediaElement.prototype.play = function () {
        this.dispatchEvent(new Event('playing'));
        return Promise.resolve();
      };
    });

    await page.goto('/');

    const player = page.getByLabel('Original live audio');
    await expect(player).toHaveAttribute('src', '/api/audio/hls/default/index.m3u8');
    await expect(player).not.toHaveAttribute('autoplay', '');

    await page.getByRole('button', { name: 'Start Listening' }).click();
    await expect(page.getByRole('button', { name: 'Start Listening' })).toBeHidden();
  });

  test('shows a retry action when the browser rejects playback', async ({ page }) => {
    await page.addInitScript(() => {
      HTMLMediaElement.prototype.play = () => Promise.reject(new DOMException('Not allowed', 'NotAllowedError'));
    });

    await page.goto('/');
    await page.getByRole('button', { name: 'Start Listening' }).click();

    await expect(page.getByRole('alert')).toContainText('Safari blocked playback');
    await expect(page.getByRole('button', { name: 'Retry Listening' })).toBeVisible();
  });
});
