async function setServerUrl(url) {
  url = url.trim().replace(/\/$/, '');

  if (!url.match(/^https?:\/\//)) {
    throw new Error('Invalid URL: must start with http:// or https://');
  }

  await browser.storage.local.set({ serverUrl: url });
}

async function getServerUrl() {
  const { serverUrl } = await browser.storage.local.get('serverUrl');
  return serverUrl || '';
}