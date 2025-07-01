const axios = require('axios');

class MailHogHelper {
  constructor(baseUrl = 'http://localhost:8026') {
    this.baseUrl = baseUrl;
    this.apiUrl = `${baseUrl}/api/v2`;
  }

  async clearEmails() {
    try {
      await axios.delete(`${this.baseUrl}/api/v1/messages`);
      return true;
    } catch (error) {
      console.error('Failed to clear emails:', error.message);
      return false;
    }
  }

  async waitForEmail(to, subjectContains, timeoutMs = 10000) {
    const startTime = Date.now();
    
    while (Date.now() - startTime < timeoutMs) {
      const email = await this.getEmail(to, subjectContains);
      if (email) {
        return email;
      }
      await this.sleep(1000);
    }
    
    throw new Error(`Email to ${to} with subject containing "${subjectContains}" not received within ${timeoutMs}ms`);
  }

  async getEmail(to, subjectContains) {
    try {
      const response = await axios.get(`${this.apiUrl}/messages`);
      const messages = response.data.items || [];

      for (const message of messages) {
        const recipients = message.To || [];
        const matchingRecipient = recipients.find(recipient => 
          `${recipient.Mailbox}@${recipient.Domain}` === to
        );

        if (matchingRecipient) {
          const subjects = message.Content.Headers.Subject || [];
          const matchingSubject = subjects.find(subject => 
            subject.includes(subjectContains)
          );

          if (matchingSubject) {
            return {
              id: message.ID,
              subject: matchingSubject,
              body: message.Content.Body,
              to: to,
              from: `${message.From.Mailbox}@${message.From.Domain}`
            };
          }
        }
      }

      return null;
    } catch (error) {
      console.error('Failed to get email:', error.message);
      return null;
    }
  }

  extractConfirmationLink(emailBody) {
    const confirmRegex = /http:\/\/[^\/]+\/api\/confirm\/([a-zA-Z0-9-]+)/;
    const match = emailBody.match(confirmRegex);
    return match ? match[0] : null;
  }

  extractUnsubscribeLink(emailBody) {
    const unsubscribeRegex = /http:\/\/[^\/]+\/api\/unsubscribe\/([a-zA-Z0-9-]+)/;
    const match = emailBody.match(unsubscribeRegex);
    return match ? match[0] : null;
  }

  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

module.exports = MailHogHelper;
