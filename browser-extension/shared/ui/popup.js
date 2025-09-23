let currentTab = 'quick';

document.addEventListener('DOMContentLoaded', async () => {
  const isAuthenticated = await checkAuth().catch(() => false);

  if (isAuthenticated) {
    showMainScreen();
  } else {
    showLoginScreen();
  }

  setupEventListeners();
});

function setupEventListeners() {
  document.getElementById('login-btn').addEventListener('click', handleLogin);
  document.getElementById('logout-btn').addEventListener('click', handleLogout);
  document.getElementById('create-quick-btn').addEventListener('click', handleCreateQuickCard);
  document.getElementById('create-article-btn').addEventListener('click', handleCreateArticleCard);
  document.getElementById('extract-btn').addEventListener('click', handleExtractArticle);

  document.querySelectorAll('.tab').forEach(tab => {
    tab.addEventListener('click', () => {
      switchTab(tab.dataset.tab);
    });
  });
}

async function handleLogin() {
  const serverUrl = document.getElementById('server-url').value;
  const email = document.getElementById('email').value;
  const password = document.getElementById('password').value;
  const errorEl = document.getElementById('login-error');

  errorEl.classList.add('hidden');

  try {
    await setServerUrl(serverUrl);
    await login(email, password);
    showMainScreen();
  } catch (error) {
    errorEl.textContent = error.message;
    errorEl.classList.remove('hidden');
  }
}

async function handleLogout() {
  await logout();
  showLoginScreen();
}

async function handleCreateQuickCard() {
  const title = document.getElementById('quick-title').value;
  const body = document.getElementById('quick-body').value;

  if (!title || !body) {
    showError('Please fill in both title and content');
    return;
  }

  try {
    await createCard({ title, body, card_type: 'standard' });
    showSuccess('Card created successfully!');
    document.getElementById('quick-title').value = '';
    document.getElementById('quick-body').value = '';
  } catch (error) {
    showError(error.message);
  }
}

async function handleCreateArticleCard() {
  const title = document.getElementById('article-title').value;
  const body = document.getElementById('article-body').value;
  const url = document.getElementById('article-url').value;

  if (!title || !body) {
    showError('Please extract an article first');
    return;
  }

  const bodyWithSource = `${body}\n\n---\nSource: ${url}`;

  try {
    await createCard({ title, body: bodyWithSource, card_type: 'article' });
    showSuccess('Article card created successfully!');
    document.getElementById('article-preview').classList.add('hidden');
  } catch (error) {
    showError(error.message);
  }
}

async function handleExtractArticle() {
  try {
    const tabs = await browser.tabs.query({ active: true, currentWindow: true });
    const tab = tabs[0];

    await browser.tabs.executeScript(tab.id, {
      file: '../content/Readability.js'
    });

    await browser.tabs.executeScript(tab.id, {
      file: '../content/extractor.js'
    });

    const response = await browser.tabs.sendMessage(tab.id, { type: 'EXTRACT_ARTICLE' });

    if (!response.success) {
      showError(response.error);
      return;
    }

    document.getElementById('article-title').value = response.title;
    document.getElementById('article-author').value = response.byline;
    document.getElementById('article-body').value = response.textContent;
    document.getElementById('article-url').value = response.metadata.url;
    document.getElementById('article-preview').classList.remove('hidden');

    showSuccess('Article extracted successfully!');
  } catch (error) {
    showError('Failed to extract article: ' + error.message);
  }
}

function switchTab(tabName) {
  currentTab = tabName;

  document.querySelectorAll('.tab').forEach(tab => {
    tab.classList.toggle('active', tab.dataset.tab === tabName);
  });

  document.getElementById('quick-tab').classList.toggle('hidden', tabName !== 'quick');
  document.getElementById('article-tab').classList.toggle('hidden', tabName !== 'article');

  hideMessages();
}

function showLoginScreen() {
  document.getElementById('login-screen').classList.remove('hidden');
  document.getElementById('main-screen').classList.add('hidden');

  getServerUrl().then(url => {
    if (url) {
      document.getElementById('server-url').value = url;
    }
  });
}

function showMainScreen() {
  document.getElementById('login-screen').classList.add('hidden');
  document.getElementById('main-screen').classList.remove('hidden');
}

function showError(message) {
  const errorEl = document.getElementById('error');
  errorEl.textContent = message;
  errorEl.classList.remove('hidden');
  document.getElementById('success').classList.add('hidden');
}

function showSuccess(message) {
  const successEl = document.getElementById('success');
  successEl.textContent = message;
  successEl.classList.remove('hidden');
  document.getElementById('error').classList.add('hidden');

  setTimeout(() => {
    successEl.classList.add('hidden');
  }, 3000);
}

function hideMessages() {
  document.getElementById('error').classList.add('hidden');
  document.getElementById('success').classList.add('hidden');
}