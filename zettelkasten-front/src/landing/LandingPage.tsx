import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import landingImage from "../assets/landing.png";
import { Footer } from "./Footer";
import { setDocumentTitle } from "../utils/title";
import { LandingHeader } from "./LandingHeader";
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
  faqs,
  faqSection,
  builtByContent,
  testimonialsSection,
} from "../data/landingContent";

// Component imports
import { HeroSection } from "./components/HeroSection";
import { FeaturesSection } from "./components/FeaturesSection";
import { PricingSection } from "./components/PricingSection";
import { VideoSection } from "./components/VideoSection";
import { PersonasSection } from "./components/PersonasSection";
import { FAQSection } from "./components/FAQSection";
import { BuiltBySection } from "./components/BuiltBySection";
import { TestimonialsSection } from "./components/TestimonialsSection";

function LandingPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedFeature, setExpandedFeature] = useState<string | null>(null);

  function handleSignUp() {
    navigate("/app");
  }

  async function handleSubmit() {
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

          {/* 1. Hero */}
          <HeroSection
            hero={heroSection}
            onSignUp={handleSignUp}
            landingImage={landingImage}
          />

          {/* 2. Personas — compact strip, no heading */}
          <PersonasSection personas={personas} />

          {/* 3. Video — show the product early */}
          <VideoSection video={videoSection} onCtaClick={handleSignUp} />

          {/* 4. Features */}
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

          {/* 5. Pricing — right after features, while intent is high */}
          <PricingSection
            tiers={pricingTiers}
            onNavigate={(route) => navigate(route)}
            sectionTitle={pricingSection.title}
            sectionDescription={pricingSection.description}
          />

          {/* 6. Community / Open Source */}
          <TestimonialsSection
            testimonials={[]}
            sectionTitle={testimonialsSection.title}
            sectionDescription={testimonialsSection.description}
          />

          {/* 7. FAQ */}
          <FAQSection
            faqs={faqs}
            sectionTitle={faqSection.title}
            sectionDescription={faqSection.description}
          />

          {/* 8. Built By + Newsletter combined */}
          <div className="mt-12">
            <BuiltBySection
              content={builtByContent}
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
          </div>

          <Footer />
        </div>
      </div>
    </div>
  );
}

export default LandingPage;
