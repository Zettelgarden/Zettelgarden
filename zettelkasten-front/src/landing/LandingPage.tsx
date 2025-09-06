import React, { useState } from "react";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Link } from "react-router-dom";
import { motion } from "framer-motion";
import landingImage from "../assets/landing.png";
import { Footer } from "./Footer";
import { setDocumentTitle } from "../utils/title";
import { LandingHeader } from "./LandingHeader";
import { RecentBlogPosts } from "./RecentBlogPosts";
import { addToMailingList } from "../api/users";

const zettel_env = import.meta.env.VITE_ENV;

// Add RSS Icon component
const RssIcon = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="24"
    height="24"
    viewBox="0 0 24 24"
    fill="currentColor"
    className="w-6 h-6"
  >
    <path d="M6.18 15.64a2.18 2.18 0 0 1 2.18 2.18C8.36 19 7.38 20 6.18 20C5 20 4 19 4 17.82a2.18 2.18 0 0 1 2.18-2.18M4 4.44A15.56 15.56 0 0 1 19.56 20h-2.83A12.73 12.73 0 0 0 4 7.27V4.44m0 5.66a9.9 9.9 0 0 1 9.9 9.9h-2.83A7.07 7.07 0 0 0 4 12.93V10.1z" />
  </svg>
);

function LandingPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);

  function handleSignUp() {
    navigate("/app");
  }

  async function handleSubmit() {
    console.log(email);
    addToMailingList(email);
    setSubmitted(true);
  }

  useEffect(() => {
    setDocumentTitle();
  }, []);

  //  const subscriptionEnabled = import.meta.env.VITE_FEATURE_SUBSCRIPTION === "true";
  const subscriptionEnabled = false

  return (
    <div className="min-h-screen bg-gradient-to-b from-white to-green-50">
      <div className="w-full py-2 mx-auto max-w-screen-xl flex items-center px-4 sm:px-6 lg:px-8">
        <div className="w-full">
          <LandingHeader />

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8 }}
            className="flex flex-col lg:flex-row gap-12 items-center mt-8">
            <div className="lg:w-6/12 space-y-6">
              <h1 className="text-4xl md:text-5xl font-bold text-gray-900 leading-tight">
                Plant Your Thoughts, Cultivate Your Ideas
              </h1>
              <p className="text-xl text-gray-600 leading-relaxed">
                Zettelgarden is an open-source personal knowledge management system
                that preserves human insight while leveraging modern technology.
                Built on zettelkasten principles, it helps you develop and maintain
                your own understanding of the world.
              </p>

              <button
                onClick={handleSignUp}
                className="px-8 py-4 bg-green-600 text-white rounded-lg font-semibold text-lg hover:bg-green-700 transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:-translate-y-1">
                Get Started with Zettelgarden
              </button>
            </div>
            <motion.div
              className="lg:w-6/12"
              whileHover={{ scale: 1.02 }}
              transition={{ duration: 0.2 }}>
              <img
                src={landingImage}
                alt="Zettelgarden interface preview"
                className="w-full rounded-xl shadow-2xl"
              />
            </motion.div>
          </motion.div>



          <div id="features" className="py-24 space-y-24">
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6 }}
              className="grid md:grid-cols-2 gap-12">
              <div className="space-y-4">
                <h2 className="text-2xl font-bold text-gray-900">Thoughtful AI Integration</h2>
                <p className="text-gray-600 leading-relaxed">
                  While other tools rush to automate everything with LLMs, Zettelgarden
                  takes a measured approach. AI features are designed to augment your
                  thinking process, not replace it. Our summarization pipeline delivers both
                  high‑level insights through executive summaries and detailed evidence‑driven
                  reference summaries with theses, arguments, and facts—helping you see the big
                  picture without losing important nuance.
                </p>
              </div>
              <div className="space-y-4">
                <h2 className="text-2xl font-bold text-gray-900">Human-Centric Knowledge Organization</h2>
                <p className="text-gray-600 leading-relaxed">
                  Create and connect atomic notes that reflect your understanding, not
                  just store information. Built on time-tested zettelkasten principles,
                  Zettelgarden helps you develop genuine insights rather than just
                  collecting automated summaries.
                </p>
              </div>
              <div className="space-y-4">
                <h2 className="text-2xl font-bold text-gray-900">Built for Scale</h2>
                <p className="text-gray-600 leading-relaxed">
                  Whether you're managing personal notes or building a company
                  knowledge base, Zettelgarden is designed to grow with you.
                  Powerful linking and organization features help maintain clarity
                  even as your knowledge base expands.
                </p>
              </div>
              <div className="space-y-4">
                <h2 className="text-2xl font-bold text-gray-900">Structured Summaries</h2>
                <p className="text-gray-600 leading-relaxed">
                  Transform dense articles, podcasts, or research into clear two‑part outputs:
                  concise executive summaries for decision‑makers and detailed reference summaries
                  with theses, ranked arguments, and verifiable facts for researchers.
                </p>
              </div>
              <div className="space-y-4">
                <h2 className="text-2xl font-bold text-gray-900">Research‑Grade Analysis</h2>
                <p className="text-gray-600 leading-relaxed">
                  Automatically extracts core claims, supports them with evidence, and ranks
                  arguments by importance. Summaries stay academically precise and fact driven,
                  providing clarity while preserving nuance.
                </p>
              </div>
              <div className="space-y-4">
                <h2 className="text-2xl font-bold text-gray-900">Open Source and Transparent</h2>
                <p className="text-gray-600 leading-relaxed">
                  Zettelgarden is built in the open, using TypeScript and Go. Your knowledge belongs to you—no
                  vendor lock‑in, no black boxes, just clean, efficient knowledge management.
                </p>
              </div>
            </motion.div>
          </div>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6 }}
            className="mt-24 mb-12"
          >
            <h2 className="text-3xl font-bold text-center mb-8">See Zettelgarden in Action</h2>
            <div className="relative w-full" style={{ paddingBottom: '56.25%' }}>
              <iframe
                src="https://www.youtube.com/embed/0kSAhX2R7eM"
                title="Zettelgarden Demo"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
                className="absolute top-0 left-0 w-full h-full rounded-xl shadow-2xl"
              ></iframe>
            </div>
          </motion.div>

          {subscriptionEnabled && (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6 }}
              className="py-24 text-center"
            >
              <h2 className="text-3xl font-bold mb-6">Simple, Transparent Pricing</h2>
              <p className="text-gray-600 mb-12 max-w-2xl mx-auto">
                Upgrade to Zettelgarden Pro to unlock advanced AI summarization, fact extraction,
                and early access to new features.
              </p>

              <div className="flex flex-col md:flex-row gap-8 justify-center items-stretch flex-wrap">
                <div className="bg-white shadow-md rounded-lg p-6 w-full max-w-sm flex flex-col">
                  <h3 className="text-xl font-semibold text-indigo-700 mb-2">Free</h3>
                  <p className="text-gray-700 mb-4">$0 / forever</p>
                  <ul className="text-left mb-6 space-y-2">
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Create and Organize Cards</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Knowledge Linking and Organization</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Manage your Todos With Your Cards</li>
                  </ul>
                  <button
                    onClick={() => navigate("/app")}
                    className="mt-auto w-full bg-green-600 text-white px-4 py-3 rounded-lg font-medium hover:bg-green-700 transition-colors"
                  >
                    Get Started Free
                  </button>
                </div>
                <div className="bg-white shadow-md rounded-lg p-6 w-full max-w-sm flex flex-col">
                  <h3 className="text-xl font-semibold text-indigo-700 mb-2">Monthly</h3>
                  <p className="text-gray-700 mb-4">$10 / month</p>
                  <ul className="text-left mb-6 space-y-2">
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Create and Organize Cards</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Knowledge Linking and Organization</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Manage your Todos With Your Cards</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> AI-Powered Entity and Fact Extraction</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Card Summarization and Analysis</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Early Access To New Features</li>
                  </ul>
                  <button
                    onClick={() => navigate("/subscribe")}
                    className="mt-auto w-full bg-indigo-600 text-white px-4 py-3 rounded-lg font-medium hover:bg-indigo-700 transition-colors"
                  >
                    Choose Monthly
                  </button>
                </div>
                <div className="bg-white shadow-md rounded-lg p-6 w-full max-w-sm flex flex-col">
                  <h3 className="text-xl font-semibold text-indigo-700 mb-2">Annual</h3>
                  <p className="text-gray-700 mb-1">$100 / year <span className="text-green-600">(Save 20%)</span></p>
                  <ul className="text-left mb-6 space-y-2">
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Create and Organize Cards</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Knowledge Linking and Organization</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Manage your Todos With Your Cards</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> AI-Powered Entity and Fact Extraction</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Card Summarization and Analysis</li>
                    <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Early Access To New Features</li>
                  </ul>
                  <button
                    onClick={() => navigate("/subscribe")}
                    className="mt-auto w-full bg-indigo-600 text-white px-4 py-3 rounded-lg font-medium hover:bg-indigo-700 transition-colors"
                  >
                    Choose Annual
                  </button>
                </div>
              </div>
            </motion.div>
          )}

          <RecentBlogPosts />

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="py-16 bg-green-50 rounded-2xl px-8 text-center">
            <h2 className="text-2xl font-bold mb-6">Stay Updated</h2>
            <p className="text-gray-600 mb-8 max-w-2xl mx-auto">
              Stay updated with Zettelgarden's development. Sign up for occasional
              updates about new features and releases.
            </p>
            {!submitted ? (
              <div className="flex flex-col sm:flex-row gap-4 justify-center items-center max-w-md mx-auto">
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="Enter your email"
                  className="w-full px-4 py-3 rounded-lg border border-gray-300 focus:ring-2 focus:ring-green-500 focus:border-transparent"
                  required
                />
                <button
                  onClick={handleSubmit}
                  className="w-full sm:w-auto px-6 py-3 bg-green-600 text-white rounded-lg font-semibold hover:bg-green-700 transition-colors duration-200">
                  Sign Up
                </button>
              </div>
            ) : (
              <p className="text-green-600 font-semibold">Thank you for signing up!</p>
            )}
          </motion.div>

          <Footer />
        </div>
      </div>
    </div>
  );
}

export default LandingPage;
