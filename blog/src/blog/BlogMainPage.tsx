import React, { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { BlogPostMeta } from "./models";
import { getAllPosts } from "./utils";

export function BlogMainPage() {
  const [posts, setPosts] = useState<BlogPostMeta[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getAllPosts()
      .then(setPosts)
      .finally(() => setIsLoading(false));
  }, []);

  return (
    <main className="mt-8 px-4">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Blog Posts</h1>
        <a
          href="/blog/rss.xml"
          className="text-orange-500 hover:text-orange-600 flex items-center gap-2"
          title="RSS Feed"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="currentColor"
            className="w-5 h-5"
          >
            <path d="M6.18 15.64a2.18 2.18 0 0 1 2.18 2.18C8.36 19 7.38 20 6.18 20C5 20 4 19 4 17.82a2.18 2.18 0 0 1 2.18-2.18M4 4.44A15.56 15.56 0 0 1 19.56 20h-2.83A12.73 12.73 0 0 0 4 7.27V4.44m0 5.66a9.9 9.9 0 0 1 9.9 9.9h-2.83A7.07 7.07 0 0 0 4 12.93V10.1z"/>
          </svg>
        </a>
      </div>

      {isLoading ? (
        <div>Loading posts...</div>
      ) : (
        <div className="grid gap-6">
          {posts.map((post) => (
            <article
              key={post.slug}
              className="border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-shadow"
            >
              <Link to={`/blog/${post.slug}`}>
                <h2 className="text-2xl font-semibold mb-2">{post.title}</h2>
                <div className="text-gray-600">
                  {new Date(post.date).toLocaleDateString("en-US", {
                    year: "numeric",
                    month: "long",
                    day: "numeric",
                  })}
                </div>
                {post.excerpt && (
                  <p className="mt-2 text-gray-600">{post.excerpt}</p>
                )}
              </Link>
            </article>
          ))}
        </div>
      )}

      {!isLoading && posts.length === 0 && (
        <div className="text-center text-gray-600">No blog posts found.</div>
      )}
    </main>
  );
}
