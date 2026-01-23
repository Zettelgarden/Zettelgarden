import React from "react";
import { motion } from "framer-motion";
import type { PricingTier } from "../../types/landing";

interface PricingSectionProps {
  tiers: PricingTier[];
  onNavigate: (route: string) => void;
  sectionTitle: string;
  sectionDescription: string;
}

function PricingCard({
  tier,
  onNavigate,
}: {
  tier: PricingTier;
  onNavigate: (route: string) => void;
}) {
  const buttonColorClass =
    tier.buttonColor === "green"
      ? "bg-green-600 hover:bg-green-700"
      : "bg-indigo-600 hover:bg-indigo-700";

  return (
    <div className="bg-white shadow-lg rounded-xl p-6 w-full max-w-sm flex flex-col border border-modern-slate-100 hover:shadow-xl transition-shadow duration-300">
      <h3 className="text-xl font-display font-semibold text-modern-indigo-700 mb-2">
        {tier.name}
      </h3>
      <p className="text-gray-700 mb-1">{tier.price}</p>
      {tier.subtitle && (
        <p className="text-sm text-green-600 mb-3">{tier.subtitle}</p>
      )}
      <ul className="text-left mb-6 space-y-2">
        {tier.features.map((feature, index) => (
          <li key={index} className="flex items-center">
            <span className="text-green-600 mr-2">✓</span>{" "}
            {feature.highlight ? <strong>{feature.text}</strong> : feature.text}
          </li>
        ))}
      </ul>
      <button
        onClick={() => onNavigate(tier.route)}
        className={`mt-auto w-full ${buttonColorClass} text-white px-4 py-3 rounded-lg font-medium transition-colors`}
        aria-label={`Choose ${tier.name} plan: ${tier.price}`}
      >
        {tier.buttonText}
      </button>
    </div>
  );
}

export function PricingSection({
  tiers,
  onNavigate,
  sectionTitle,
  sectionDescription,
}: PricingSectionProps) {
  return (
    <motion.div
      id="pricing"
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="py-24 text-center"
    >
      <h2 className="text-3xl font-display font-bold mb-6 text-modern-slate-900">
        {sectionTitle}
      </h2>
      <p className="font-body text-modern-slate-600 mb-12 max-w-2xl mx-auto">
        {sectionDescription}
      </p>

      <div className="flex flex-col md:flex-row gap-8 justify-center items-stretch flex-wrap">
        {tiers.map((tier) => (
          <PricingCard key={tier.id} tier={tier} onNavigate={onNavigate} />
        ))}
      </div>
    </motion.div>
  );
}
