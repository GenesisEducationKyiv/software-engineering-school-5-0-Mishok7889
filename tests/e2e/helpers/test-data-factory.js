class TestDataFactory {
  static generateUniqueEmail() {
    const timestamp = Date.now();
    const random = Math.floor(Math.random() * 1000);
    return `test-${timestamp}-${random}@example.com`;
  }

  static generateTestUser() {
    return {
      email: this.generateUniqueEmail(),
      city: 'London',
      frequency: 'daily'
    };
  }

  static getTestCities() {
    return [
      { name: 'London', expectedTemp: 15, expectedHumidity: 76, expectedDesc: 'Partly cloudy' },
      { name: 'Paris', expectedTemp: 18, expectedHumidity: 68, expectedDesc: 'Clear' },
      { name: 'Berlin', expectedTemp: 12, expectedHumidity: 82, expectedDesc: 'Overcast' }
    ];
  }

  static getInvalidEmails() {
    return [
      'invalid-email',
      'test@',
      '@domain.com',
      'test.domain.com',
      ''
    ];
  }

  static getValidationTestCases() {
    return {
      emptyEmail: { email: '', city: 'London', frequency: 'daily' },
      invalidEmail: { email: 'invalid', city: 'London', frequency: 'daily' },
      emptyCity: { email: 'test@example.com', city: '', frequency: 'daily' },
      invalidFrequency: { email: 'test@example.com', city: 'London', frequency: 'weekly' }
    };
  }
}

module.exports = TestDataFactory;
