import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { BlogPostMeta } from '../blog/models';
import { getAllPosts } from '../blog/utils';

export function RecentBlogPosts() {
  const [posts, setPosts] = useState<BlogPostMeta[]>([]);

  useEffect(() => {
    const fetchPosts = async () => {
      const allPosts = await getAllPosts();
      setPosts(allPosts.slice(0, 2)); // Get only the two most recent posts
    };
    fetchPosts();
  }, []);

  if (posts.length === 0) return null;

  return (
    <motion.div 
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="py-24">
      <h2 className="text-3xl font-bold text-modern-slate-900 mb-8">Latest from Our Blog</h2>
      <div className="grid md:grid-cols-2 gap-8">
        {posts.map((post) => (
          <Link 
            key={post.slug} 
            to={`/blog/${post.slug}`}
            className="group">
            <motion.article 
              className="bg-white p-6 rounded-xl shadow-md hover:shadow-xl transition-shadow duration-200"
              whileHover={{ y: -4 }}
              transition={{ duration: 0.2 }}>
              <h3 className="text-xl font-semibold text-modern-slate-900 group-hover:text-modern-emerald-600 transition-colors duration-200">
                {post.title}
              </h3>
              <p className="text-sm text-modern-slate-500 mt-2">
                {new Date(post.date).toLocaleDateString('en-US', {
                  year: 'numeric',
                  month: 'long',
                  day: 'numeric'
                })} • {post.author}
              </p>
              {post.excerpt && (
                <p className="text-modern-slate-600 mt-4 line-clamp-3">
                  {post.excerpt}
                </p>
              )}
              {post.tags && post.tags.length > 0 && (
                <div className="flex gap-2 mt-4 flex-wrap">
                  {post.tags.map(tag => (
                    <span 
                      key={tag}
                      className="px-2 py-1 bg-modern-emerald-50 text-modern-emerald-600 rounded-full text-sm">
                      {tag}
                    </span>
                  ))}
                </div>
              )}
            </motion.article>
          </Link>
        ))}
      </div>
      <div className="text-center mt-8">
        <Link 
          to="/blog"
          className="inline-flex items-center text-modern-emerald-600 hover:text-modern-emerald-700 font-medium">
          Read more posts
          <svg 
            className="ml-2 w-4 h-4" 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24">
            <path 
              strokeLinecap="round" 
              strokeLinejoin="round" 
              strokeWidth={2} 
              d="M9 5l7 7-7 7" 
            />
          </svg>
        </Link>
      </div>
    </motion.div>
  );
} 