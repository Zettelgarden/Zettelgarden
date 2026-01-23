import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import landingImage from "../assets/landing.png";
import { Footer } from "./Footer";
import { setDocumentTitle } from "../utils/title";
import { LandingHeader } from "./LandingHeader";
import { RecentBlogPosts } from "./RecentBlogPosts";
import { addToMailingList } from "../api/users";

// Data imports
import {
  heroSection,
  features,
  pricingTiers,
  videoSection,
  newsletterSection,
  featuresSection,
  pricingSection,
} from "../data/landingContent";

// Component imports
import { HeroSection } from "./components/HeroSection";
import { FeaturesSection } from "./components/FeaturesSection";
import { PricingSection } from "./components/PricingSection";
import { VideoSection } from "./components/VideoSection";
import { NewsletterSection } from "./components/NewsletterSection";

const zettel_env = import.meta.env.VITE_ENV;

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
  const subscriptionEnabled = false;

  return (
    <div className="min-h-screen bg-gradient-to-br from-white via-modern-slate-50 to-modern-emerald-50">
      <div className="w-full py-2 mx-auto max-w-screen-xl flex items-center px-4 sm:px-6 lg:px-8">
        <div className="w-full">
          <LandingHeader />

          <HeroSection
            hero={heroSection}
            onSignUp={handleSignUp}
            scrollY={scrollY}
            landingImage={landingImage}
          />

          <FeaturesSection
            features={features}
            expandedFeature={expandedFeature}
            onExpandFeature={setExpandedFeature}
            hoveredCard={hoveredCard}
            onHoverCard={setHoveredCard}
            sectionTitle={featuresSection.title}
            sectionDescription={featuresSection.description}
          />

          <div className="mt-24 mb-12">
            <VideoSection video={videoSection} />
          </div>

          <PricingSection
            tiers={pricingTiers}
            onNavigate={(route) => navigate(route)}
            sectionTitle={pricingSection.title}
            sectionDescription={pricingSection.description}
          />

          <RecentBlogPosts />

          <NewsletterSection
            newsletter={newsletterSection}
            email={email}
            onEmailChange={setEmail}
            submitted={submitted}
            onSubmit={handleSubmit}
          />

          <Footer />
        </div>
      </div>
    </div>
  );
}

export default LandingPage;
