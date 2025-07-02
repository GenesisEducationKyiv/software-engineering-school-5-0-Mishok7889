const { test, expect } = require('@playwright/test');
const MailHogHelper = require('../helpers/mailhog-helper');
const TestDataFactory = require('../helpers/test-data-factory');

test.describe('Subscription Workflow', () => {
  let mailhog;

  test.beforeAll(async () => {
    mailhog = new MailHogHelper();
  });

  test.beforeEach(async ({ page }) => {
    await mailhog.clearEmails();
    await page.goto('/');
  });

  test('should complete full subscription workflow', async ({ page }) => {
    // Arrange
    const testUser = TestDataFactory.generateTestUser();
    
    // Act 1 - Fill subscription form using correct IDs
    await page.fill('#email', testUser.email);
    await page.fill('#city', testUser.city);
    await page.selectOption('#frequency', testUser.frequency);
    
    // Submit subscription
    await page.click('button[type="submit"]');
    
    // Assert 1 - Success message should appear with correct ID
    await expect(page.locator('#success-message')).toBeVisible();
    await expect(page.locator('#success-message')).toContainText('Thank you for subscribing');
    
    // Act 2 - Wait for confirmation email
    const confirmationEmail = await mailhog.waitForEmail(
      testUser.email, 
      'Confirm your weather subscription'
    );
    
    // Assert 2 - Confirmation email received
    expect(confirmationEmail).toBeTruthy();
    expect(confirmationEmail.subject).toContain('Confirm your weather subscription');
    
    // Act 3 - Extract confirmation link and visit it
    const confirmationLink = mailhog.extractConfirmationLink(confirmationEmail.body);
    expect(confirmationLink).toBeTruthy();
    
    // Visit confirmation link - this is an API endpoint, not a page
    const response = await page.request.get(confirmationLink);
    expect(response.ok()).toBeTruthy();
    
    const confirmationData = await response.json();
    expect(confirmationData.message).toContain('Subscription confirmed');
    
    // Act 4 - Wait for welcome email
    const welcomeEmail = await mailhog.waitForEmail(
      testUser.email,
      'Welcome to Weather Updates'
    );
    
    // Assert 4 - Welcome email received
    expect(welcomeEmail).toBeTruthy();
    expect(welcomeEmail.subject).toContain('Welcome to Weather Updates');
  });

  test('should handle subscription form validation', async ({ page }) => {
    // Test empty email - HTML5 validation should prevent submission
    await page.fill('#email', '');
    await page.fill('#city', 'London');
    await page.selectOption('#frequency', 'daily');
    
    // Try to submit - HTML5 validation should prevent it
    await page.click('button[type="submit"]');
    
    // Check if email field has required attribute and is invalid
    const emailInput = page.locator('#email');
    await expect(emailInput).toHaveAttribute('required');
    
    // Test invalid email format
    await page.fill('#email', 'invalid-email');
    await page.click('button[type="submit"]');
    
    // HTML5 validation should catch invalid email
    const isInvalid = await emailInput.evaluate(el => !el.validity.valid);
    expect(isInvalid).toBeTruthy();
    
    // Test server-side validation with invalid frequency via direct API call
    // This will fail at the Gin binding level due to struct validation tags
    const response = await page.request.post('/api/subscribe', {
      form: {
        email: 'test@example.com',
        city: 'London',
        frequency: 'invalid-frequency'
      }
    });
    
    expect(response.status()).toBe(400);
    const errorData = await response.json();
    expect(errorData.error).toContain('invalid request format');
  });

  test('should allow same email to subscribe to different cities', async ({ page }) => {
    const testUser = TestDataFactory.generateTestUser();
    
    // First subscription to London
    await page.fill('#email', testUser.email);
    await page.fill('#city', 'London');
    await page.selectOption('#frequency', 'daily');
    await page.click('button[type="submit"]');
    await expect(page.locator('#success-message')).toBeVisible();
    
    // Subscribe to Paris with same email - should succeed
    await page.goto('/');
    await page.fill('#email', testUser.email);
    await page.fill('#city', 'Paris');
    await page.selectOption('#frequency', 'hourly');
    await page.click('button[type="submit"]');
    
    // Should show success message (allowing multiple cities)
    await expect(page.locator('#success-message')).toBeVisible();
    await expect(page.locator('#success-message')).toContainText('Thank you for subscribing');
  });

  test('should handle already subscribed email', async ({ page }) => {
    const testUser = TestDataFactory.generateTestUser();
    
    // First subscription
    await page.fill('#email', testUser.email);
    await page.fill('#city', testUser.city);
    await page.selectOption('#frequency', testUser.frequency);
    await page.click('button[type="submit"]');
    await expect(page.locator('#success-message')).toBeVisible();
    
    // Confirm first subscription
    const confirmationEmail = await mailhog.waitForEmail(testUser.email, 'Confirm');
    const confirmationLink = mailhog.extractConfirmationLink(confirmationEmail.body);
    const confirmResponse = await page.request.get(confirmationLink);
    expect(confirmResponse.ok()).toBeTruthy();
    
    // Try to subscribe again with same email AND same city - should get conflict
    await page.goto('/');
    await page.fill('#email', testUser.email);
    await page.fill('#city', testUser.city); // Same city this time
    await page.selectOption('#frequency', 'hourly'); // Different frequency
    await page.click('button[type="submit"]');
    
    // Should show already subscribed error
    await expect(page.locator('#error-message')).toBeVisible();
    await expect(page.locator('#error-message')).toContainText('already subscribed');
  });

  test('should handle subscription update before confirmation', async ({ page }) => {
    const testUser = TestDataFactory.generateTestUser();
    
    // First subscription
    await page.fill('#email', testUser.email);
    await page.fill('#city', testUser.city);
    await page.selectOption('#frequency', 'hourly');
    await page.click('button[type="submit"]');
    await expect(page.locator('#success-message')).toBeVisible();
    
    // Update subscription before confirming - reload page first
    await page.goto('/');
    await page.fill('#email', testUser.email);
    await page.fill('#city', testUser.city);
    await page.selectOption('#frequency', 'daily');
    await page.click('button[type="submit"]');
    
    // Should allow update and show success
    await expect(page.locator('#success-message')).toBeVisible();
    await expect(page.locator('#success-message')).toContainText('Thank you for subscribing');
  });

  test('should handle unsubscribe workflow', async ({ page }) => {
    const testUser = TestDataFactory.generateTestUser();
    
    // Subscribe and confirm
    await page.fill('#email', testUser.email);
    await page.fill('#city', testUser.city);
    await page.selectOption('#frequency', testUser.frequency);
    await page.click('button[type="submit"]');
    await expect(page.locator('#success-message')).toBeVisible();
    
    // Get confirmation email and confirm
    const confirmationEmail = await mailhog.waitForEmail(testUser.email, 'Confirm');
    const confirmationLink = mailhog.extractConfirmationLink(confirmationEmail.body);
    const confirmResponse = await page.request.get(confirmationLink);
    expect(confirmResponse.ok()).toBeTruthy();
    
    // Get welcome email and extract unsubscribe link
    const welcomeEmail = await mailhog.waitForEmail(testUser.email, 'Welcome');
    const unsubscribeLink = mailhog.extractUnsubscribeLink(welcomeEmail.body);
    expect(unsubscribeLink).toBeTruthy();
    
    // Use unsubscribe link
    const unsubscribeResponse = await page.request.get(unsubscribeLink);
    expect(unsubscribeResponse.ok()).toBeTruthy();
    
    const unsubscribeData = await unsubscribeResponse.json();
    expect(unsubscribeData.message).toContain('Unsubscribed successfully');
    
    // Should receive unsubscribe confirmation email
    const unsubscribeConfirmEmail = await mailhog.waitForEmail(
      testUser.email, 
      'unsubscribed from weather updates'
    );
    expect(unsubscribeConfirmEmail).toBeTruthy();
  });
});
