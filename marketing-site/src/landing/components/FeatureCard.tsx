import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ChevronDownIcon } from "../../assets/icons/ChevronDownIcon";
import type { Feature } from "../../types/landing";

interface FeatureCardProps {
  feature: Feature;
  isExpanded: boolean;
  onToggle: () => void;
}

export function FeatureCard({
  feature,
  isExpanded,
  onToggle,
}: FeatureCardProps) {
  const cardRef = React.useRef<HTMLDivElement>(null);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      onToggle();
    }
    if (e.key === " ") {
      e.preventDefault();
      onToggle();
    }
  };

  return (
    <motion.div
      ref={cardRef}
      layout
      className="space-y-4 cursor-pointer bg-white/50 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300"
      whileHover={{ y: -4, scale: 1.02 }}
      onClick={onToggle}
      onKeyDown={handleKeyDown}
      role="button"
      tabIndex={0}
      aria-expanded={isExpanded}
      aria-controls={`feature-details-${feature.id}`}
    >
      <div className="flex items-center gap-3">
        <span className="flex items-center justify-center w-10 h-10 rounded-lg bg-modern-emerald-50 text-modern-emerald-600" aria-hidden="true">
          {React.createElement(feature.icon, { className: "w-5 h-5" })}
        </span>
        <h2 className="text-2xl font-display font-bold text-modern-slate-900">
          {feature.title}
        </h2>
        <motion.div
          animate={{ rotate: isExpanded ? 180 : 0 }}
          transition={{ duration: 0.2 }}
          className="ml-auto"
          aria-hidden="true"
        >
          <ChevronDownIcon className="w-5 h-5 text-modern-slate-600" />
        </motion.div>
      </div>

      <p className="font-body text-modern-slate-600 leading-relaxed">
        {feature.summary}
      </p>

      <AnimatePresence>
        {isExpanded && (
          <motion.div
            id={`feature-details-${feature.id}`}
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="overflow-hidden"
            role="region"
            aria-label={`${feature.title} details`}
          >
            <div className="pt-4 border-t border-modern-slate-200">
              <p className="font-body text-modern-slate-700 leading-relaxed">
                {feature.details}
              </p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
