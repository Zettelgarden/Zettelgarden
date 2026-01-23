import React, { useRef, useEffect } from "react";
import { motion } from "framer-motion";
import type { NewsletterContent } from "../../types/landing";

interface NewsletterSectionProps {
  newsletter: NewsletterContent;
  email: string;
  onEmailChange: (email: string) => void;
  submitted: boolean;
  onSubmit: () => void;
  loading: boolean;
  error: string | null;
}

export function NewsletterSection({
  newsletter,
  email,
  onEmailChange,
  submitted,
  onSubmit,
  loading,
  error,
}: NewsletterSectionProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Focus input when error appears
  useEffect(() => {
    if (error && inputRef.current) {
      inputRef.current.focus();
    }
  }, [error]);

  // Focus button on successful submit
  useEffect(() => {
    if (submitted && buttonRef.current) {
      buttonRef.current.focus();
    }
  }, [submitted]);

  const handleSubmit = () => {
    if (!loading) {
      onSubmit();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !loading && !submitted) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      className="py-16 bg-gradient-to-br from-modern-emerald-50 to-modern-slate-50 rounded-2xl px-8 text-center border border-modern-emerald-100"
      role="region"
      aria-labelledby="newsletter-title"
    >
      <h2
        id="newsletter-title"
        className="text-2xl font-display font-bold mb-6 text-modern-slate-900"
      >
        {newsletter.title}
      </h2>
      <p className="font-body text-modern-slate-600 mb-8 max-w-2xl mx-auto">
        {newsletter.description}
      </p>
      {!submitted ? (
        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center max-w-md mx-auto">
          <div className="w-full">
            <label htmlFor="newsletter-email" className="sr-only">
              Email address
            </label>
            <input
              ref={inputRef}
              id="newsletter-email"
              type="email"
              value={email}
              onChange={(e) => onEmailChange(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={newsletter.inputPlaceholder}
              className="w-full px-4 py-3 rounded-lg border border-modern-slate-300 focus:ring-2 focus:ring-modern-emerald-500 focus:border-transparent font-body"
              disabled={loading}
              aria-invalid={error ? "true" : "false"}
              aria-describedby={error ? "newsletter-error" : undefined}
              required
            />
            {error && (
              <p
                id="newsletter-error"
                className="text-red-600 text-sm mt-2 text-left"
                role="alert"
              >
                {error}
              </p>
            )}
          </div>
          <button
            ref={buttonRef}
            onClick={handleSubmit}
            disabled={loading || !email}
            className="w-full sm:w-auto px-6 py-3 bg-modern-emerald-600 text-white rounded-lg font-body font-semibold hover:bg-modern-emerald-700 transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            aria-label={newsletter.buttonText}
          >
            {loading ? newsletter.buttonLoadingText : newsletter.buttonText}
          </button>
        </div>
      ) : (
        <p
          className="text-modern-emerald-600 font-body font-semibold"
          role="status"
          aria-live="polite"
        >
          {newsletter.successMessage}
        </p>
      )}
    </motion.div>
  );
}
