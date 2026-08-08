import React from "react";
import { Routes, Route } from "react-router-dom";
import { BlogApp } from "./blog/BlogApp";

function App() {
  return (
    <div>
      <Routes>
        {/* Kept under /blog/* so post URLs and the RSS feed path
            (/blog/rss.xml) match the pre-extraction layout. */}
        <Route path="/blog/*" element={<BlogApp />} />
        <Route path="*" element={<BlogApp />} />
      </Routes>
    </div>
  );
}

export default App;
