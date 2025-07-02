const { test, expect } = require('@playwright/test');

test.describe('Debug Subscription API', () => {
  test('should debug subscription API directly', async ({ page }) => {
    // Test the API endpoint directly first
    const response = await page.request.post('/api/subscribe', {
      form: {
        email: 'debug@example.com',
        city: 'London',
        frequency: 'daily'
      }
    });

    console.log('API Response Status:', response.status());
    console.log('API Response Body:', await response.text());
    
    // Also test via the form to see what happens
    await page.goto('/');
    
    // Fill form
    await page.fill('#email', 'debug@example.com');
    await page.fill('#city', 'London');
    await page.selectOption('#frequency', 'daily');
    
    // Listen for console errors
    page.on('console', msg => console.log('Browser console:', msg.text()));
    
    // Listen for network requests
    page.on('response', response => {
      if (response.url().includes('/api/subscribe')) {
        console.log('Form submission response status:', response.status());
      }
    });
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Wait a bit and check what's visible
    await page.waitForTimeout(2000);
    
    // Check both success and error messages
    const successVisible = await page.locator('#success-message').isVisible();
    const errorVisible = await page.locator('#error-message').isVisible();
    
    console.log('Success message visible:', successVisible);
    console.log('Error message visible:', errorVisible);
    
    if (errorVisible) {
      const errorText = await page.locator('#error-message').textContent();
      console.log('Error message text:', errorText);
    }
    
    // Take a screenshot for debugging
    await page.screenshot({ path: 'debug-subscription-form.png' });
  });
});
