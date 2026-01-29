import React, { useState } from "react";
import { Link } from "react-router-dom";
import { motion, AnimatePresence } from "framer-motion";
import logo from "../assets/logo.png";
import { useAuth } from "../contexts/AuthContext";

export function LandingHeader() {
  const { isAuthenticated } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <motion.div
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.5 }}
      className="relative w-full py-4">

      {/* Main header */}
      <div className="flex items-center justify-between w-full">
        {/* Logo */}
        <Link to="/" className="flex items-center gap-2 sm:gap-3 group flex-shrink-0">
          <motion.img
            whileHover={{ rotate: 10 }}
            src={logo}
            alt="Company Logo"
            className="w-8 h-8 sm:w-10 sm:h-10 rounded-md shadow-md"
          />
          <span className="text-xl sm:text-2xl font-display font-bold bg-gradient-to-r from-modern-emerald-600 to-modern-emerald-800 bg-clip-text text-transparent">
            Zettelgarden
          </span>
        </Link>

        {/* Desktop Navigation */}
        <div className="hidden md:flex items-center space-x-8">
          <a
            href="/#features"
            className="font-body text-modern-slate-600 hover:text-modern-slate-900 font-medium transition-colors duration-200">
            Features
          </a>
          <a
            href="/#pricing"
            className="font-body text-modern-slate-600 hover:text-modern-slate-900 font-medium transition-colors duration-200">
            Pricing
          </a>
          <a
            href="https://nsavage.substack.com"
            className="font-body text-modern-slate-600 hover:text-modern-slate-900 font-medium transition-colors duration-200">
            Blog
          </a>
        </div>

        {/* Desktop Login Button */}
        <Link
          to="/app"
          className="hidden sm:block px-4 sm:px-6 py-2 bg-modern-emerald-600 text-white rounded-lg font-body font-medium hover:bg-modern-emerald-700 transition-colors duration-200 shadow-sm hover:shadow-md flex-shrink-0">
          {isAuthenticated ? "Go To App" : "Get Started Free"}
        </Link>

        {/* Mobile Menu Button */}
        <button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="md:hidden p-2 text-modern-slate-600 hover:text-modern-slate-900 flex-shrink-0"
          aria-label="Toggle mobile menu">
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d={mobileMenuOpen ? "M6 18L18 6M6 6l12 12" : "M4 6h16M4 12h16M4 18h16"}
            />
          </svg>
        </button>
      </div>

      {/* Mobile Menu */}
      <AnimatePresence>
        {mobileMenuOpen && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2 }}
            className="md:hidden mt-4 py-4 border-t border-modern-slate-200 bg-white/90 backdrop-blur-sm rounded-lg shadow-lg">
            <div className="flex flex-col space-y-4">
              <a
                href="/#features"
                onClick={() => setMobileMenuOpen(false)}
                className="px-4 py-2 font-body text-modern-slate-600 hover:text-modern-slate-900 font-medium transition-colors duration-200">
                Features
              </a>
              <a
                href="/#pricing"
                onClick={() => setMobileMenuOpen(false)}
                className="px-4 py-2 font-body text-modern-slate-600 hover:text-modern-slate-900 font-medium transition-colors duration-200">
                Pricing
              </a>
              <a
                href="https://nsavage.substack.com"
                onClick={() => setMobileMenuOpen(false)}
                className="px-4 py-2 font-body text-modern-slate-600 hover:text-modern-slate-900 font-medium transition-colors duration-200">
                Blog
              </a>
              <Link
                to="/app"
                onClick={() => setMobileMenuOpen(false)}
                className="mx-4 px-6 py-2 bg-modern-emerald-600 text-white rounded-lg font-body font-medium hover:bg-modern-emerald-700 transition-colors duration-200 shadow-sm text-center">
                {isAuthenticated ? "Go To App" : "Get Started Free"}
              </Link>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
