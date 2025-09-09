import React from "react";
import { Link } from "react-router-dom";
import { motion } from "framer-motion";
import logo from "../assets/logo.png";
import { useAuth } from "../contexts/AuthContext";

export function LandingHeader() {
  const { isAuthenticated } = useAuth();
  
  return (
    <motion.div 
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.5 }}
      className="flex items-center w-full py-4">
      <Link to="/" className="flex items-center gap-3 group">
        <motion.img 
          whileHover={{ rotate: 10 }}
          src={logo} 
          alt="Company Logo" 
          className="w-10 h-10 rounded-md shadow-md" 
        />
        <span className="text-2xl font-bold bg-gradient-to-r from-green-600 to-green-800 bg-clip-text text-transparent">
          Zettelgarden
        </span>
      </Link>
      
      <div className="flex-grow flex justify-center space-x-8">
        <a
          href="/#features"
          className="text-gray-600 hover:text-gray-900 font-medium transition-colors duration-200">
          Features
        </a>
        <a
          href="/#pricing"
          className="text-gray-600 hover:text-gray-900 font-medium transition-colors duration-200">
          Pricing
        </a>
        <a
          href="/blog"
          className="text-gray-600 hover:text-gray-900 font-medium transition-colors duration-200">
          Blog
        </a>
      </div>
      
      <Link 
        to="/app"
        className="px-6 py-2 bg-green-600 text-white rounded-lg font-medium hover:bg-green-700 transition-colors duration-200 shadow-sm hover:shadow-md">
        {isAuthenticated ? "Go To App" : "Login"}
      </Link>
    </motion.div>
  );
}
