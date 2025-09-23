async function login(email, password) {
  const response = await browser.runtime.sendMessage({
    type: 'API_CALL',
    payload: {
      endpoint: '/api/login',
      method: 'POST',
      body: { email, password },
      requiresAuth: false
    }
  });

  if (response.error) {
    throw new Error(response.error);
  }

  await browser.storage.local.set({ authToken: response.token });
  return response;
}

async function checkAuth() {
  const response = await browser.runtime.sendMessage({
    type: 'API_CALL',
    payload: {
      endpoint: '/api/auth',
      method: 'GET',
      requiresAuth: true
    }
  });

  return !response.error;
}

async function logout() {
  await browser.storage.local.remove(['authToken']);
}

async function getAuthToken() {
  const { authToken } = await browser.storage.local.get('authToken');
  return authToken;
}