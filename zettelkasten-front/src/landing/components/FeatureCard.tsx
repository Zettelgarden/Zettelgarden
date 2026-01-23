import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ChevronDownIcon } from "../../assets/icons/ChevronDownIcon";
import type { Feature } from "../../types/landing";

interface FeatureCardProps {
  feature: Feature;
  isExpanded: boolean;
  onToggle: () => void;
  isHovered: boolean;
  onHover: (hovered: boolean) => void;
}

export function FeatureCard({
  feature,
  isExpanded,
  onToggle,
  isHovered,
  onHover,
}: FeatureCardProps) {
  return (
    <motion.div
      layout
      className="space-y-4 cursor-pointer bg-white/50 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300"
      whileHover={{ y: -4, scale: 1.02 }}
      onClick={onToggle}
      onHoverStart={() => onHover(true)}
      onHoverEnd={() => onHover(false)}
    >
      <div className="flex items-center gap-3">
        <span className="text-2xl">{feature.icon}</span>
        <h2 className="text-2xl font-display font-bold text-modern-slate-900">
          {feature.title}
        </h2>
        <motion.div
          animate={{ rotate: isExpanded ? 180 : 0 }}
          transition={{ duration: 0.2 }}
          className="ml-auto"
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
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="overflow-hidden"
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
