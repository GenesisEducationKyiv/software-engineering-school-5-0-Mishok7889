const { test, expect } = require('@playwright/test');

test.describe('Weather API Direct Testing', () => {
  test('should return weather data for valid city via API', async ({ request }) => {
    // Test weather API endpoint directly
    const response = await request.get('/api/weather?city=London');
    expect(response.ok()).toBeTruthy();
    
    const data = await response.json();
    expect(data).toHaveProperty('Temperature');
    expect(data).toHaveProperty('Humidity');
    expect(data).toHaveProperty('Description');
    expect(data.Temperature).toBe(15.0);
    expect(data.Humidity).toBe(76.0);
    expect(data.Description).toBe('Partly cloudy');
  });

  test('should return error for invalid city via API', async ({ request }) => {
    const response = await request.get('/api/weather?city=NonExistentCity');
    expect(response.status()).toBe(500);
    
    const data = await response.json();
    expect(data).toHaveProperty('error');
    expect(data.error).toBe('Failed to get weather data');
  });

  test('should return error for missing city parameter', async ({ request }) => {
    const response = await request.get('/api/weather');
    expect(response.status()).toBe(400);
    
    const data = await response.json();
    expect(data).toHaveProperty('error');
    expect(data.error).toBe('city parameter is required');
  });

  test('should test multiple cities via API', async ({ request }) => {
    const cities = [
      { name: 'London', expectedTemp: 15.0, expectedHumidity: 76.0, expectedDesc: 'Partly cloudy' },
      { name: 'Paris', expectedTemp: 18.0, expectedHumidity: 68.0, expectedDesc: 'Clear' },
      { name: 'Berlin', expectedTemp: 12.0, expectedHumidity: 82.0, expectedDesc: 'Overcast' }
    ];
    
    for (const city of cities) {
      const response = await request.get(`/api/weather?city=${city.name}`);
      expect(response.ok()).toBeTruthy();
      
      const data = await response.json();
      expect(data.Temperature).toBe(city.expectedTemp);
      expect(data.Humidity).toBe(city.expectedHumidity);
      expect(data.Description).toBe(city.expectedDesc);
    }
  });
});
