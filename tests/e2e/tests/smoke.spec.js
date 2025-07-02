const { test, expect } = require('@playwright/test');

test.describe('Application Smoke Tests', () => {
  test('should load homepage successfully', async ({ page }) => {
    await page.goto('/');
    
    // Basic page load verification
    await expect(page).toHaveTitle('Weather Forecast Subscription');
    
    // Check for main elements
    await expect(page.locator('h1')).toBeVisible();
    await expect(page.locator('h1')).toContainText('Weather Updates Subscription');
    await expect(page.locator('#subscription-form')).toBeVisible();
    
    // Check form fields exist
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#city')).toBeVisible();
    await expect(page.locator('#frequency')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should have working debug endpoint', async ({ page }) => {
    const response = await page.request.get('/api/debug');
    expect(response.ok()).toBeTruthy();
    
    const data = await response.json();
    expect(data).toHaveProperty('database');
    expect(data).toHaveProperty('weatherAPI');
    expect(data).toHaveProperty('smtp');
  });

  test('should handle 404 pages', async ({ page }) => {
    const response = await page.goto('/nonexistent-page');
    expect(response.status()).toBe(404);
  });
});
