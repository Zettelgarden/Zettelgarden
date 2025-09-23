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

  if (!response.access_token) {
    throw new Error('No access token received from server');
  }

  await browser.storage.local.set({ authToken: response.access_token });
  return response;
}

async function checkAuth() {
  try {
    const response = await browser.runtime.sendMessage({
      type: 'API_CALL',
      payload: {
        endpoint: '/api/auth',
        method: 'GET',
        requiresAuth: true
      }
    });

    // If there's an error in the response, auth failed
    if (response.error) {
      return false;
    }

    return true;
  } catch (error) {
    console.error('Auth check failed:', error);
    return false;
  }
}

async function logout() {
  await browser.storage.local.remove(['authToken']);
}

async function getAuthToken() {
  const { authToken } = await browser.storage.local.get('authToken');
  return authToken;
}