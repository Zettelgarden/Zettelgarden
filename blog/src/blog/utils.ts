import matter from 'gray-matter';
import { BlogPost, BlogPostMeta } from './models';

interface PostFrontmatter {
  title: string;
  date: string;
  author: string;
  excerpt?: string;
  tags?: string[];  // Added tags property
}

export async function getAllPosts(): Promise<BlogPostMeta[]> {
  const posts: BlogPostMeta[] = [];
  const postFiles = import.meta.glob('./posts/*.md', { as: 'raw' });
  console.log(postFiles)
  
  for (const path in postFiles) {
    const file = await postFiles[path]();
    const slug = path.replace(/(\.\/posts\/|\.md)/g, '');
    
    const { data } = matter(file);
    const frontmatter = data as PostFrontmatter;
    
    posts.push({
      slug,
      author: frontmatter.author,
      title: frontmatter.title,
      date: frontmatter.date,
      excerpt: frontmatter.excerpt,
      tags: frontmatter.tags  // Include tags in the response
    });
  }

  return posts.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
}

export async function getPost(slug: string): Promise<BlogPost | null> {
  try {
    const content = await import(`./posts/${slug}.md?raw`);
    
    const { data, content: markdownContent } = matter(content.default);
    const frontmatter = data as PostFrontmatter;

    return {
      slug,
      author: frontmatter.author,
      content: markdownContent,
      title: frontmatter.title,
      date: frontmatter.date,
      excerpt: frontmatter.excerpt,
      tags: frontmatter.tags  // Include tags in the response
    };
  } catch {
    return null;
  }
}
