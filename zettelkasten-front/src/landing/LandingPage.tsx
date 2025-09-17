import React, { useState } from "react";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Link } from "react-router-dom";
import { motion, AnimatePresence } from "framer-motion";
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
  const [expandedFeature, setExpandedFeature] = useState<string | null>(null);
  const [hoveredCard, setHoveredCard] = useState<string | null>(null);
  const [scrollY, setScrollY] = useState(0);

  useEffect(() => {
    const handleScroll = () => setScrollY(window.scrollY);
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

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

  const features = [
    {
      id: "ai-integration",
      title: "Thoughtful AI Integration",
      summary: "While other tools rush to automate everything with LLMs, Zettelgarden takes a measured approach.",
      details: "AI features are designed to augment your thinking process, not replace it. Our summarization pipeline delivers both high‑level insights through executive summaries and detailed evidence‑driven reference summaries with theses, arguments, and facts—helping you see the big picture without losing important nuance.",
      icon: "🧠"
    },
    {
      id: "human-centric",
      title: "Human-Centric Knowledge Organization",
      summary: "Create and connect atomic notes that reflect your understanding, not just store information.",
      details: "Built on time-tested zettelkasten principles, Zettelgarden helps you develop genuine insights rather than just collecting automated summaries. Every connection you make strengthens your personal knowledge graph.",
      icon: "🌱"
    },
    {
      id: "scale",
      title: "Built for Scale",
      summary: "Whether you're managing personal notes or building a company knowledge base, Zettelgarden is designed to grow with you.",
      details: "Powerful linking and organization features help maintain clarity even as your knowledge base expands. Advanced search capabilities and hierarchical organization ensure you can always find what you need.",
      icon: "📈"
    },
    {
      id: "summaries",
      title: "Structured Summaries",
      summary: "Transform dense articles, podcasts, or research into clear two‑part outputs.",
      details: "Concise executive summaries for decision‑makers and detailed reference summaries with theses, ranked arguments, and verifiable facts for researchers. Each summary preserves the original context while making information actionable.",
      icon: "📋"
    },
    {
      id: "analysis",
      title: "Research‑Grade Analysis",
      summary: "Automatically extracts core claims, supports them with evidence, and ranks arguments by importance.",
      details: "Summaries stay academically precise and fact driven, providing clarity while preserving nuance. Our analysis pipeline identifies key themes, cross-references claims, and maintains citation integrity.",
      icon: "🔬"
    },
    {
      id: "open-source",
      title: "Open Source and Transparent",
      summary: "Zettelgarden is built in the open, using TypeScript and Go.",
      details: "Your knowledge belongs to you—no vendor lock‑in, no black boxes, just clean, efficient knowledge management. Full source code is available on GitHub with comprehensive documentation.",
      icon: "🔓"
    }
  ];

  const FeatureCard = ({ feature, isExpanded, onToggle, isHovered, onHover }: {
    feature: typeof features[0],
    isExpanded: boolean,
    onToggle: () => void,
    isHovered: boolean,
    onHover: (hovered: boolean) => void
  }) => (
    <motion.div
      layout
      className="space-y-4 cursor-pointer bg-white/50 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300"
      whileHover={{ y: -4, scale: 1.02 }}
      onClick={onToggle}
      onHoverStart={() => onHover(true)}
      onHoverEnd={() => onHover(false)}
    >
      <div className="flex items-center gap-3">
        <span className="text-2xl">{feature.icon}</span>
        <h2 className="text-2xl font-display font-bold text-modern-slate-900">{feature.title}</h2>
        <motion.div
          animate={{ rotate: isExpanded ? 180 : 0 }}
          transition={{ duration: 0.2 }}
          className="ml-auto"
        >
          <svg className="w-5 h-5 text-modern-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </motion.div>
      </div>

      <p className="font-body text-modern-slate-600 leading-relaxed">
        {feature.summary}
      </p>

      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="overflow-hidden"
          >
            <div className="pt-4 border-t border-modern-slate-200">
              <p className="font-body text-modern-slate-700 leading-relaxed">
                {feature.details}
              </p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );

  return (
    <div className="min-h-screen bg-gradient-to-br from-white via-modern-slate-50 to-modern-emerald-50">
      <div className="w-full py-2 mx-auto max-w-screen-xl flex items-center px-4 sm:px-6 lg:px-8">
        <div className="w-full">
          <LandingHeader />

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8 }}
            className="relative flex flex-col lg:flex-row gap-12 items-center mt-8">

            {/* Floating decorative elements with parallax */}
            <motion.div
              className="absolute -top-4 -left-4 w-20 h-20 bg-gradient-to-br from-modern-emerald-200 to-modern-emerald-300 rounded-full opacity-60 blur-sm"
              animate={{
                y: [0, -10, 0],
                scale: [1, 1.1, 1]
              }}
              style={{
                transform: `translateY(${scrollY * 0.2}px)`
              }}
              transition={{
                duration: 4,
                repeat: Infinity,
                ease: "easeInOut"
              }}
            />
            <motion.div
              className="absolute top-1/3 -right-8 w-16 h-16 bg-gradient-to-br from-modern-indigo-200 to-modern-indigo-300 rounded-full opacity-40 blur-sm"
              animate={{
                y: [0, 15, 0],
                x: [0, -5, 0]
              }}
              style={{
                transform: `translateY(${scrollY * -0.15}px)`
              }}
              transition={{
                duration: 6,
                repeat: Infinity,
                ease: "easeInOut",
                delay: 2
              }}
            />

            <div className="lg:w-6/12 space-y-6 relative z-10">
              <motion.h1
                className="text-4xl md:text-6xl font-display font-bold text-modern-slate-900 leading-tight tracking-tight"
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.8, delay: 0.2 }}
              >
                Plant Your Thoughts, <span className="text-modern-emerald-600 relative">
                  Cultivate
                  <motion.div
                    className="absolute -bottom-1 left-0 w-full h-1 bg-gradient-to-r from-modern-emerald-400 to-modern-emerald-600"
                    initial={{ scaleX: 0 }}
                    animate={{ scaleX: 1 }}
                    transition={{ duration: 0.8, delay: 1 }}
                  />
                </span> Your Ideas
              </motion.h1>

              <motion.p
                className="text-xl font-body text-modern-slate-600 leading-relaxed"
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.8, delay: 0.4 }}
              >
                Zettelgarden is an open-source personal knowledge management system
                that preserves human insight while leveraging modern technology.
                Built on zettelkasten principles, it helps you develop and maintain
                your own understanding of the world.
              </motion.p>

              <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.8, delay: 0.6 }}
              >
                <motion.button
                  onClick={handleSignUp}
                  whileHover={{
                    scale: 1.05,
                    y: -2,
                    boxShadow: "0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)"
                  }}
                  whileTap={{ scale: 0.98 }}
                  className="px-8 py-4 bg-gradient-to-r from-modern-emerald-600 to-modern-emerald-700 text-white rounded-xl font-body font-semibold text-lg hover:from-modern-emerald-700 hover:to-modern-emerald-800 transition-all duration-300 shadow-lg relative overflow-hidden group">
                  <span className="relative z-10">Get Started with Zettelgarden</span>
                  <motion.div
                    className="absolute inset-0 bg-gradient-to-r from-modern-emerald-700 to-modern-emerald-800"
                    initial={{ x: "-100%" }}
                    whileHover={{ x: "0%" }}
                    transition={{ duration: 0.3 }}
                  />
                </motion.button>
              </motion.div>
            </div>

            <motion.div
              className="lg:w-6/12 relative"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.8, delay: 0.4 }}
              whileHover={{
                scale: 1.02,
                rotateY: 2,
                rotateX: 1
              }}
            >
              <motion.div
                className="absolute -inset-4 bg-gradient-to-r from-modern-emerald-400 to-modern-indigo-400 rounded-xl opacity-20 blur-lg"
                animate={{
                  scale: [1, 1.05, 1],
                  opacity: [0.2, 0.3, 0.2]
                }}
                transition={{
                  duration: 3,
                  repeat: Infinity,
                  ease: "easeInOut"
                }}
              />
              <img
                src={landingImage}
                alt="Zettelgarden interface preview"
                className="relative w-full rounded-xl shadow-2xl border border-modern-slate-200"
              />
            </motion.div>
          </motion.div>



          <div id="features" className="py-24 space-y-8">
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6 }}
              className="text-center mb-12"
            >
              <h2 className="text-4xl font-display font-bold text-modern-slate-900 mb-4">
                Features that <span className="text-modern-emerald-600">Work for You</span>
              </h2>
              <p className="text-xl font-body text-modern-slate-600 max-w-3xl mx-auto">
                Click on any feature to learn more about how Zettelgarden can enhance your knowledge management workflow.
              </p>
            </motion.div>

            <motion.div
              initial={{ opacity: 0 }}
              whileInView={{ opacity: 1 }}
              viewport={{ once: true }}
              transition={{ duration: 0.8, staggerChildren: 0.1 }}
              className="grid md:grid-cols-2 lg:grid-cols-3 gap-6"
            >
              {features.map((feature, index) => (
                <motion.div
                  key={feature.id}
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ duration: 0.6, delay: index * 0.1 }}
                >
                  <FeatureCard
                    feature={feature}
                    isExpanded={expandedFeature === feature.id}
                    onToggle={() => setExpandedFeature(expandedFeature === feature.id ? null : feature.id)}
                    isHovered={hoveredCard === feature.id}
                    onHover={(hovered) => setHoveredCard(hovered ? feature.id : null)}
                  />
                </motion.div>
              ))}
            </motion.div>
          </div>


          <motion.div
            id="pricing"
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6 }}
            className="py-24 text-center"
          >
            <h2 className="text-3xl font-display font-bold mb-6 text-modern-slate-900">Simple, Transparent Pricing</h2>
            <p className="font-body text-modern-slate-600 mb-12 max-w-2xl mx-auto">
              Upgrade to Zettelgarden Pro to unlock advanced AI summarization, fact extraction,
              and early access to new features.
            </p>

            <div className="flex flex-col md:flex-row gap-8 justify-center items-stretch flex-wrap">
              <div className="bg-white shadow-lg rounded-xl p-6 w-full max-w-sm flex flex-col border border-modern-slate-100 hover:shadow-xl transition-shadow duration-300">
                <h3 className="text-xl font-display font-semibold text-modern-indigo-700 mb-2">Free</h3>
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
              <div className="bg-white shadow-lg rounded-xl p-6 w-full max-w-sm flex flex-col border border-modern-slate-100 hover:shadow-xl transition-shadow duration-300">
                <h3 className="text-xl font-display font-semibold text-modern-indigo-700 mb-2">Monthly</h3>
                <p className="text-gray-700 mb-1">$10 / month</p>
                <p className="text-sm text-green-600 mb-3">30-day free trial included</p>
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
              <div className="bg-white shadow-lg rounded-xl p-6 w-full max-w-sm flex flex-col border border-modern-slate-100 hover:shadow-xl transition-shadow duration-300">
                <h3 className="text-xl font-display font-semibold text-modern-indigo-700 mb-2">Annual</h3>
                <p className="text-gray-700 mb-1">$100 / year <span className="text-green-600">(Save 20%)</span></p>
                <p className="text-sm text-green-600 mb-3">30-day free trial included</p>
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
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6 }}
              className="mt-24 mb-12"
            >
              <h2 className="text-3xl font-display font-bold text-center mb-8 text-modern-slate-900">See Zettelgarden in Action</h2>
              <div className="relative w-full" style={{ paddingBottom: '56.25%' }}>
                <iframe
                  src="https://www.youtube.com/embed/0kSAhX2R7eM"
                  title="Zettelgarden Demo"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowFullScreen
                  className="absolute top-0 left-0 w-full h-full rounded-xl shadow-2xl"
                ></iframe>
              </div>
            </motion.div>           </motion.div>

          <RecentBlogPosts />

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="py-16 bg-gradient-to-br from-modern-emerald-50 to-modern-slate-50 rounded-2xl px-8 text-center border border-modern-emerald-100">
            <h2 className="text-2xl font-display font-bold mb-6 text-modern-slate-900">Stay Updated</h2>
            <p className="font-body text-modern-slate-600 mb-8 max-w-2xl mx-auto">
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
                  className="w-full px-4 py-3 rounded-lg border border-modern-slate-300 focus:ring-2 focus:ring-modern-emerald-500 focus:border-transparent font-body"
                  required
                />
                <button
                  onClick={handleSubmit}
                  className="w-full sm:w-auto px-6 py-3 bg-modern-emerald-600 text-white rounded-lg font-body font-semibold hover:bg-modern-emerald-700 transition-colors duration-200">
                  Sign Up
                </button>
              </div>
            ) : (
              <p className="text-modern-emerald-600 font-body font-semibold">Thank you for signing up!</p>
            )}
          </motion.div>

          <Footer />
        </div>
      </div>
    </div>
  );
}

export default LandingPage;
