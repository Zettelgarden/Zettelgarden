import React from "react";
import { Routes, Route } from "react-router-dom";
import { BlogMainPage } from "./BlogMainPage";
import { BlogPostComponent } from "./BlogPost";
import { generateRssFeed } from "./rss";
import { LandingHeader } from "../components/LandingHeader";
import { Footer } from "../components/Footer";

export function BlogApp() {
  // Handle RSS feed request
  React.useEffect(() => {
    const handleRssFeed = async () => {
      if (window.location.pathname === "/blog/rss.xml") {
        const feed = await generateRssFeed();
        const blob = new Blob([feed], { type: "application/xml" });
        const url = URL.createObjectURL(blob);
        window.location.href = url;
      }
    };
    handleRssFeed();
  }, []);

  return (
    <div className="min-h-screen flex flex-col">
      <div className="w-full py-2 mx-auto max-w-screen-xl flex-grow">
        <LandingHeader />
        <Routes>
          <Route path="/" element={<BlogMainPage />} />
          <Route path="/:slug" element={<BlogPostComponent />} />
        </Routes>
      </div>
      <Footer />
    </div>
  );
}
