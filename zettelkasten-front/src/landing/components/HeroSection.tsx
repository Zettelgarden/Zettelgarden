import React from "react";
import { motion } from "framer-motion";
import { Check } from "lucide-react";
import type { HeroContent } from "../../types/landing";

interface HeroSectionProps {
  hero: HeroContent;
  onSignUp: () => void;
  landingImage: string;
}

export function HeroSection({ hero, onSignUp, landingImage }: HeroSectionProps) {
  return (
    <section className="relative flex flex-col lg:flex-row gap-12 items-center mt-8 py-12 lg:py-20">
      <div className="lg:w-6/12 space-y-6 relative z-10">
        <motion.h1
          className="text-4xl md:text-6xl font-display font-extrabold text-modern-slate-900 leading-[1.1] tracking-tight"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          {hero.title}{" "}
          <span className="text-modern-emerald-600">
            {hero.titleHighlighted}
          </span>
          {hero.titleSuffix || ", Not Just Notes"}
        </motion.h1>

        <motion.p
          className="text-lg font-body text-modern-slate-600 leading-relaxed max-w-xl"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.1 }}
        >
          {hero.description}
        </motion.p>

        {hero.outcomes && (
          <motion.ul
            className="space-y-2"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
          >
            {hero.outcomes.map((outcome, index) => (
              <li
                key={index}
                className="flex items-center gap-2 text-base font-body text-modern-slate-700"
              >
                <span className="flex items-center justify-center w-5 h-5 rounded-full bg-modern-emerald-100 text-modern-emerald-600 flex-shrink-0">
                  <Check className="w-3 h-3" strokeWidth={3} />
                </span>
                <span>{outcome}</span>
              </li>
            ))}
          </motion.ul>
        )}

        <motion.div
          className="flex flex-col sm:flex-row items-start sm:items-center gap-4"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.3 }}
        >
          <button
            onClick={onSignUp}
            className="px-8 py-3.5 bg-modern-emerald-600 text-white rounded-lg font-body font-semibold text-lg hover:bg-modern-emerald-700 transition-colors duration-200 shadow-lg shadow-modern-emerald-600/20"
            aria-label={hero.buttonText}
          >
            {hero.buttonText}
          </button>

          <a
            href="https://github.com/NickSavage/Zettelgarden"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 text-sm font-body text-modern-slate-500 hover:text-modern-slate-700 transition-colors"
          >
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
            </svg>
            Open source on GitHub
          </a>
        </motion.div>
      </div>

      <motion.div
        className="lg:w-6/12 relative"
        initial={{ opacity: 0, x: 40 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ duration: 0.8, delay: 0.2 }}
      >
        <div className="absolute -inset-4 bg-gradient-to-r from-modern-emerald-200 to-modern-indigo-200 rounded-2xl opacity-30 blur-2xl" />
        <img
          src={landingImage}
          alt="Zettelgarden interface preview"
          loading="lazy"
          decoding="async"
          width="1200"
          height="800"
          className="relative w-full rounded-xl shadow-2xl border border-modern-slate-200"
        />
      </motion.div>
    </section>
  );
}
