import React, { useState, useEffect } from "react";
import { setDocumentTitle } from "../utils/title";
import Markdown from "react-markdown";
import { useParams } from "react-router-dom";
import { BlogPost } from "./models";
import { getPost } from "./utils";
import { subscribeToNewsletter } from "../lib/newsletter";

import { H1, H2, H3, H4, H5, H6 } from "../components/Header";

export const BlogPostComponent: React.FC = () => {
  const { slug } = useParams();
  const [post, setPost] = useState<BlogPost | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);

  async function handleSubmit() {
    subscribeToNewsletter(email);
    setSubmitted(true);
  }

  useEffect(() => {
    const fetchPost = async () => {
      if (!slug) return;

      try {
        const postData = await getPost(slug);
        if (postData) {
          setPost(postData);
	  setDocumentTitle(postData.title);
        } else {
          setError("Post not found");
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load post");
      } finally {
        setIsLoading(false);
      }
    };

    fetchPost();
  }, [slug]);

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;
  if (!post) return <div>Post not found</div>;

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <article className="prose lg:prose-xl prose-slate mx-auto">
        {/* Header Section */}
        <header className="mb-8">
          <h2 className="text-4xl font-bold mb-4">{post.title}</h2>
          <div className="flex items-center gap-4 text-gray-600 mb-4">
            <time className="text-sm">
              {post.author + " - "}
              {new Date(post.date).toLocaleDateString("en-US", {
                year: "numeric",
                month: "long",
                day: "numeric",
              })}
            </time>
            {post.tags && (
              <div className="flex flex-wrap gap-2">
                {post.tags.map((tag) => (
                  <span
                    key={tag}
                    className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-full hover:bg-gray-200 transition-colors"
                  >
                    #{tag}
                  </span>
                ))}
              </div>
            )}
          </div>
          {post.excerpt && (
            <p className="text-xl text-gray-600 italic border-l-4 border-gray-200 pl-4">
              {post.excerpt}
            </p>
          )}
        </header>

        {/* Content Section */}
        <div
          className="prose-headings:font-bold prose-headings:text-gray-900 
                      prose-p:text-gray-700 prose-p:leading-relaxed
                      prose-a:text-blue-600 prose-a:no-underline hover:prose-a:underline
                      prose-code:bg-gray-100 prose-code:text-gray-800 prose-code:px-1 prose-code:rounded
                      prose-pre:bg-gray-900 prose-pre:text-gray-100 prose-pre:p-4 prose-pre:rounded-lg
                      prose-img:rounded-lg prose-img:shadow-md
                      prose-ul:list-disc prose-ol:list-decimal"
        >
          <Markdown>{post.content}</Markdown>

          <div>
            <p>
              Sign up for our mailing list to get notified on new blog posts:
            </p>
            {!submitted ? (
              <div>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="Enter your email"
                  required
                />
                <button onClick={handleSubmit} type="submit">
                  Sign Up
                </button>
              </div>
            ) : (
              <p>Thank you for signing up!</p>
            )}
          </div>
        </div>

        {/* Footer Section */}
        <footer className="mt-12 pt-6 border-t border-gray-200">
          <div className="flex justify-between items-center">
            <div className="text-sm text-gray-500">
              Published on {new Date(post.date).toLocaleDateString()}
            </div>
            <a href="#top" className="text-sm text-blue-600 hover:underline">
              Back to top ↑
            </a>
          </div>
        </footer>
      </article>
    </div>
  );
};
