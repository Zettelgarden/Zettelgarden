async function getNextRootId() {
  const response = await browser.runtime.sendMessage({
    type: 'API_CALL',
    payload: {
      endpoint: '/api/cards/next-root-id',
      method: 'GET',
      requiresAuth: true
    }
  });

  if (response.error) {
    throw new Error(response.error);
  }

  return response;
}

async function createCard({ title, body, url, card_type = 'article', parent_id = null, tags = [] }) {
  let card_id = "";
  if (card_type === 'article') {
    const nextIdResp = await getNextRootId();
    if (nextIdResp.error) throw new Error('Unable to fetch next ID');
    card_id = nextIdResp.new_id;
  }

  const response = await browser.runtime.sendMessage({
    type: 'API_CALL',
    payload: {
      endpoint: '/api/cards',
      method: 'POST',
      body: {
        title: title || 'Untitled',
        body: (body || '') + '\n\n#to-read #reference',
        card_id: card_id,
        link: url,
        parent_id,
        tags,
        process_entities_and_facts: true
      },
      requiresAuth: true
    }
  });

  if (response.error) {
    throw new Error(response.error);
  }

  return response;
}
