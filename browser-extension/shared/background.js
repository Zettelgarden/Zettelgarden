browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'API_CALL') {
    handleApiCall(message.payload)
      .then(sendResponse)
      .catch(error => sendResponse({ error: error.message }));
    return true;
  }
});

async function handleApiCall({ endpoint, method = 'GET', body, requiresAuth = false }) {
  const config = await browser.storage.local.get(['serverUrl', 'authToken']);

  if (!config.serverUrl) {
    throw new Error('Server URL not configured');
  }

  const headers = {
    'Content-Type': 'application/json'
  };

  if (requiresAuth) {
    if (!config.authToken) {
      throw new Error('Not authenticated');
    }
    headers['Authorization'] = `Bearer ${config.authToken}`;
  }

  const response = await fetch(`${config.serverUrl}${endpoint}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`API Error: ${response.status} - ${error}`);
  }

  return await response.json();
}