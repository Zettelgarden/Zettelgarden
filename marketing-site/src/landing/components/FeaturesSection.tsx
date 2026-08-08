import React from "react";
import { motion } from "framer-motion";
import { FeatureCard } from "./FeatureCard";
import type { Feature } from "../../types/landing";

interface FeaturesSectionProps {
  features: Feature[];
  expandedFeature: string | null;
  onExpandFeature: (id: string | null) => void;
  sectionTitle: string;
  sectionDescription: string;
  ctaText?: string;
  ctaSubtext?: string;
  onCtaClick?: () => void;
}

export function FeaturesSection({
  features,
  expandedFeature,
  onExpandFeature,
  sectionTitle,
  sectionDescription,
  ctaText,
  ctaSubtext,
  onCtaClick,
}: FeaturesSectionProps) {
  return (
    <div id="features" className="py-24 space-y-8">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6 }}
        className="text-center mb-12"
      >
        <h2 className="text-4xl font-display font-bold text-modern-slate-900 mb-4">
          {sectionTitle}
        </h2>
        <p className="text-xl font-body text-modern-slate-600 max-w-3xl mx-auto">
          {sectionDescription}
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
              onToggle={() =>
                onExpandFeature(
                  expandedFeature === feature.id ? null : feature.id
                )
              }
            />
          </motion.div>
        ))}
      </motion.div>

      {ctaText && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.3 }}
          className="mt-16 text-center"
        >
          <h3 className="text-2xl font-display font-semibold text-modern-slate-900 mb-2">
            {ctaText}
          </h3>
          {ctaSubtext && (
            <p className="text-modern-slate-600 mb-6">{ctaSubtext}</p>
          )}
          {onCtaClick && (
            <motion.button
              onClick={onCtaClick}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.98 }}
              className="px-8 py-3 bg-modern-emerald-600 text-white rounded-lg font-body font-semibold hover:bg-modern-emerald-700 transition-colors duration-200 shadow-lg"
            >
              Get Started Free
            </motion.button>
          )}
        </motion.div>
      )}
    </div>
  );
}
