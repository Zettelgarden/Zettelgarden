import React from "react";
import { motion } from "framer-motion";
import { FeatureCard } from "./FeatureCard";
import type { Feature } from "../../types/landing";

interface FeaturesSectionProps {
  features: Feature[];
  expandedFeature: string | null;
  onExpandFeature: (id: string | null) => void;
  hoveredCard: string | null;
  onHoverCard: (id: string | null) => void;
  sectionTitle: string;
  sectionDescription: string;
}

export function FeaturesSection({
  features,
  expandedFeature,
  onExpandFeature,
  hoveredCard,
  onHoverCard,
  sectionTitle,
  sectionDescription,
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
              isHovered={hoveredCard === feature.id}
              onHover={(hovered) => onHoverCard(hovered ? feature.id : null)}
            />
          </motion.div>
        ))}
      </motion.div>
    </div>
  );
}
