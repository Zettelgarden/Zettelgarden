import React from "react";
import { motion } from "framer-motion";
import { Check } from "lucide-react";
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
  const isRecommended = tier.recommended;

  return (
    <div
      className={`
        relative rounded-2xl p-6 w-full max-w-sm flex flex-col transition-shadow duration-300
        ${isRecommended
          ? "bg-white shadow-xl border-2 border-modern-emerald-500 md:scale-105 md:z-10"
          : "bg-white shadow-lg border border-modern-slate-200 hover:shadow-xl"
        }
      `}
    >
      {isRecommended && (
        <div className="absolute -top-3.5 left-1/2 -translate-x-1/2">
          <span className="px-3 py-1 bg-modern-emerald-600 text-white text-xs font-semibold rounded-full font-body">
            Best Value
          </span>
        </div>
      )}

      <h3 className={`text-xl font-display font-semibold mb-2 ${isRecommended ? "text-modern-emerald-700" : "text-modern-slate-900"}`}>
        {tier.name}
      </h3>
      <p className="text-2xl font-display font-bold text-modern-slate-900 mb-1">
        {tier.price}
      </p>
      {tier.subtitle && (
        <p className={`text-sm mb-4 ${isRecommended ? "text-modern-emerald-600" : "text-modern-slate-500"}`}>
          {tier.subtitle}
        </p>
      )}

      <ul className="text-left mb-8 space-y-3 flex-1">
        {tier.features.map((feature, index) => (
          <li key={index} className="flex items-start gap-2">
            <span className={`flex-shrink-0 mt-0.5 ${isRecommended ? "text-modern-emerald-600" : "text-modern-slate-400"}`}>
              <Check className="w-4 h-4" strokeWidth={2.5} />
            </span>
            <span className={`text-sm font-body ${feature.highlight ? "font-semibold text-modern-slate-900" : "text-modern-slate-600"}`}>
              {feature.text}
            </span>
          </li>
        ))}
      </ul>

      <button
        onClick={() => onNavigate(tier.route)}
        className={`
          w-full px-4 py-3 rounded-lg font-body font-semibold transition-all duration-200
          ${isRecommended
            ? "bg-modern-emerald-600 text-white hover:bg-modern-emerald-700 shadow-lg shadow-modern-emerald-600/20"
            : "bg-modern-slate-900 text-white hover:bg-modern-slate-800"
          }
        `}
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
    <div id="pricing" className="py-24 text-center">
      <motion.h2
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6 }}
        className="text-3xl font-display font-bold mb-6 text-modern-slate-900"
      >
        {sectionTitle}
      </motion.h2>
      <motion.p
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6, delay: 0.1 }}
        className="font-body text-modern-slate-600 mb-16 max-w-2xl mx-auto"
      >
        {sectionDescription}
      </motion.p>

      <div className="flex flex-col md:flex-row gap-6 lg:gap-8 justify-center items-center md:items-stretch">
        {tiers.map((tier, index) => (
          <motion.div
            key={tier.id}
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: index * 0.1 }}
          >
            <PricingCard tier={tier} onNavigate={onNavigate} />
          </motion.div>
        ))}
      </div>
    </div>
  );
}
