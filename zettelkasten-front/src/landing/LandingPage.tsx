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
  personas,
  personasSection,
  faqs,
  faqSection,
  builtByContent,
} from "../data/landingContent";

// Component imports
import { HeroSection } from "./components/HeroSection";
import { FeaturesSection } from "./components/FeaturesSection";
import { PricingSection } from "./components/PricingSection";
import { VideoSection } from "./components/VideoSection";
import { NewsletterSection } from "./components/NewsletterSection";
import { PersonasSection } from "./components/PersonasSection";
import { FAQSection } from "./components/FAQSection";
import { BuiltBySection } from "./components/BuiltBySection";

function LandingPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedFeature, setExpandedFeature] = useState<string | null>(null);
  const [scrollY, setScrollY] = useState(0);

  useEffect(() => {
    let ticking = false;
    const handleScroll = () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          setScrollY(window.scrollY);
          ticking = false;
        });
        ticking = true;
      }
    };
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  function handleSignUp() {
    navigate("/app");
  }

  async function handleSubmit() {
    // Basic email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!email || !emailRegex.test(email)) {
      setError("Please enter a valid email address");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await addToMailingList(email);
      setSubmitted(true);
    } catch (err) {
      setError("Something went wrong. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    setDocumentTitle();
  }, []);

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

          <PersonasSection
            personas={personas}
            sectionTitle={personasSection.title}
            sectionDescription={personasSection.description}
          />

          <FeaturesSection
            features={features}
            expandedFeature={expandedFeature}
            onExpandFeature={setExpandedFeature}
            sectionTitle={featuresSection.title}
            sectionDescription={featuresSection.description}
            ctaText={featuresSection.ctaText}
            ctaSubtext={featuresSection.ctaSubtext}
            onCtaClick={handleSignUp}
          />

          <VideoSection video={videoSection} onCtaClick={handleSignUp} />

          <PricingSection
            tiers={pricingTiers}
            onNavigate={(route) => navigate(route)}
            sectionTitle={pricingSection.title}
            sectionDescription={pricingSection.description}
          />

          <RecentBlogPosts />

          <FAQSection
            faqs={faqs}
            sectionTitle={faqSection.title}
            sectionDescription={faqSection.description}
          />

          <NewsletterSection
            newsletter={newsletterSection}
            email={email}
            onEmailChange={(value) => {
              setEmail(value);
              setError(null);
            }}
            submitted={submitted}
            onSubmit={handleSubmit}
            loading={loading}
            error={error}
          />

          <div className="mt-12">
            <BuiltBySection content={builtByContent} />
          </div>

          <Footer />
        </div>
      </div>
    </div>
  );
}

export default LandingPage;
