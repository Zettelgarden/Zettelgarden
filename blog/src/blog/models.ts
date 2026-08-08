// types/blog.ts
export interface BlogPost {
  slug: string;
  content: string;
  title: string;
  date: string;
  author: string;
  excerpt?: string;
  tags?: string[];  // Added tags property
}

export interface BlogPostMeta {
  slug: string;
  title: string;
  date: string;
  author: string;
  excerpt?: string;
  tags?: string[];  // Added tags property
}
