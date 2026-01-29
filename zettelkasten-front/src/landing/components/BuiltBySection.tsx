import React from "react";
import { motion } from "framer-motion";
import type { BuiltByContent } from "../../types/landing";

interface BuiltBySectionProps {
  content: BuiltByContent;
}

export function BuiltBySection({ content }: BuiltBySectionProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="py-16 bg-gradient-to-br from-modern-emerald-50 to-modern-slate-50 rounded-2xl px-8 border border-modern-emerald-100"
    >
      <div className="max-w-3xl mx-auto text-center">
        {content.founderPhoto && (
          <div className="mb-6 flex justify-center">
            <motion.img
              whileHover={{ scale: 1.05 }}
              src={content.founderPhoto}
              alt={content.founderName}
              className="w-24 h-24 rounded-full shadow-lg border-4 border-white"
            />
          </div>
        )}
        <h2 className="text-2xl font-display font-bold mb-4 text-modern-slate-900">
          Built with Care
        </h2>
        <p className="font-body text-modern-slate-700 leading-relaxed mb-4">
          Hi, I'm <span className="font-semibold text-modern-emerald-700">{content.founderName}</span>.
          {content.tagline}
        </p>
        {content.story && (
          <p className="font-body text-modern-slate-600 leading-relaxed mb-6">
            {content.story}
          </p>
        )}
        {content.githubUrl && (
          <motion.a
            href={content.githubUrl}
            target="_blank"
            rel="noopener noreferrer"
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.98 }}
            className="inline-flex items-center gap-2 px-6 py-3 bg-modern-slate-900 text-white rounded-lg font-body font-medium hover:bg-modern-slate-800 transition-colors duration-200"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
            </svg>
            <span>View on GitHub</span>
          </motion.a>
        )}
      </div>
    </motion.div>
  );
}
