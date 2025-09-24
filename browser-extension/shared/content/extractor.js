function extractArticle() {
  const documentClone = document.cloneNode(true);
  const article = new Readability(documentClone).parse();

  if (!article) {
    return {
      success: false,
      error: 'Could not extract article from this page'
    };
  }

  const title = article.title || document.title;
  const content = article.content || '';
  const textContent = article.textContent || '';

  const metadata = {
    url: window.location.href,
    siteName: article.siteName || new URL(window.location.href).hostname
  };

  return {
    success: true,
    title,
    content,
    textContent,
    metadata
  };
}

browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'EXTRACT_ARTICLE') {
    sendResponse(extractArticle());
  }
});