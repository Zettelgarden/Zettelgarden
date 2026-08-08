// components/BlogList.tsx
import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { BlogPostMeta } from './models';

export const BlogList: React.FC = () => {
  const [posts, setPosts] = useState<BlogPostMeta[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchPosts = async () => {
      try {
        // Implement your fetch logic here
        const response = await fetch('/api/posts');
        const data = await response.json();
        setPosts(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch posts');
      } finally {
        setIsLoading(false);
      }
    };

    fetchPosts();
  }, []);

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div className="grid gap-4">
      {posts.map((post) => (
        <div key={post.slug} className="p-4 border rounded">
          <Link to={`/blog/${post.slug}`}>
            <h2 className="text-xl font-bold">{post.title}</h2>
            {post.excerpt && <p className="text-gray-600">{post.excerpt}</p>}
            <span className="text-sm text-gray-500">{post.date}</span>
            {post.tags && (
              <div className="flex gap-2 mt-2">
                {post.tags.map((tag) => (
                  <span key={tag} className="text-sm bg-gray-100 px-2 py-1 rounded">
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </Link>
        </div>
      ))}
    </div>
  );
};
