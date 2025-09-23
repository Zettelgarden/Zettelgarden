async function createCard({ title, body, card_type = 'article', parent_id = null, tags = [] }) {
  const response = await browser.runtime.sendMessage({
    type: 'API_CALL',
    payload: {
      endpoint: '/api/cards',
      method: 'POST',
      body: {
        title,
        body,
        card_type,
        parent_id,
        tags
      },
      requiresAuth: true
    }
  });

  if (response.error) {
    throw new Error(response.error);
  }

  return response;
}