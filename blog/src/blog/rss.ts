import { Feed } from 'feed';
import { getAllPosts, getPost } from './utils';

export async function generateRssFeed(): Promise<string> {
  const posts = await getAllPosts();
  const siteURL = window.location.origin;
  
  const feed = new Feed({
    title: "Zettelgarden Blog",
    description: "Latest blog posts from Zettelgarden - Plant Your Thoughts, Cultivate Your Ideas",
    id: siteURL,
    link: siteURL,
    language: "en",
    favicon: `${siteURL}/favicon.ico`,
    copyright: `All rights reserved ${new Date().getFullYear()}`,
    updated: posts.length > 0 ? new Date(posts[0].date) : new Date(),
    feedLinks: {
      rss2: `${siteURL}/rss.xml`,
    },
  });

  // Add posts to feed
  for (const postMeta of posts) {
    const post = await getPost(postMeta.slug);
    if (!post) continue;

    feed.addItem({
      title: post.title,
      id: `${siteURL}/blog/${post.slug}`,
      link: `${siteURL}/blog/${post.slug}`,
      description: post.excerpt,
      content: post.content,
      author: [
        {
          name: post.author,
        },
      ],
      date: new Date(post.date),
    });
  }

  return feed.rss2();
} 