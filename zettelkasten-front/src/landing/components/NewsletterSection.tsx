import React from "react";
import { motion } from "framer-motion";
import type { NewsletterContent } from "../../types/landing";

interface NewsletterSectionProps {
  newsletter: NewsletterContent;
  email: string;
  onEmailChange: (email: string) => void;
  submitted: boolean;
  onSubmit: () => void;
}

export function NewsletterSection({
  newsletter,
  email,
  onEmailChange,
  submitted,
  onSubmit,
}: NewsletterSectionProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      className="py-16 bg-gradient-to-br from-modern-emerald-50 to-modern-slate-50 rounded-2xl px-8 text-center border border-modern-emerald-100"
    >
      <h2 className="text-2xl font-display font-bold mb-6 text-modern-slate-900">
        {newsletter.title}
      </h2>
      <p className="font-body text-modern-slate-600 mb-8 max-w-2xl mx-auto">
        {newsletter.description}
      </p>
      {!submitted ? (
        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center max-w-md mx-auto">
          <input
            type="email"
            value={email}
            onChange={(e) => onEmailChange(e.target.value)}
            placeholder={newsletter.inputPlaceholder}
            className="w-full px-4 py-3 rounded-lg border border-modern-slate-300 focus:ring-2 focus:ring-modern-emerald-500 focus:border-transparent font-body"
            required
          />
          <button
            onClick={onSubmit}
            className="w-full sm:w-auto px-6 py-3 bg-modern-emerald-600 text-white rounded-lg font-body font-semibold hover:bg-modern-emerald-700 transition-colors duration-200"
          >
            {newsletter.buttonText}
          </button>
        </div>
      ) : (
        <p className="text-modern-emerald-600 font-body font-semibold">
          {newsletter.successMessage}
        </p>
      )}
    </motion.div>
  );
}
