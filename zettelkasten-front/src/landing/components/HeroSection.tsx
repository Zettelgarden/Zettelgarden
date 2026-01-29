import React from "react";
import { motion } from "framer-motion";
import type { HeroContent } from "../../types/landing";

interface HeroSectionProps {
  hero: HeroContent;
  onSignUp: () => void;
  scrollY: number;
  landingImage: string;
  prefersReducedMotion?: boolean;
}

export function HeroSection({ hero, onSignUp, scrollY, landingImage, prefersReducedMotion = false }: HeroSectionProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.8 }}
      className="relative flex flex-col lg:flex-row gap-12 items-center mt-8"
    >
      {/* Floating decorative elements with parallax */}
      {!prefersReducedMotion && (
        <>
          <motion.div
            className="absolute -top-4 -left-4 w-20 h-20 bg-gradient-to-br from-modern-emerald-200 to-modern-emerald-300 rounded-full opacity-60 blur-sm"
            animate={{
              y: [0, -10, 0],
              scale: [1, 1.1, 1],
            }}
            style={{
              transform: `translateY(${scrollY * 0.2}px)`,
            }}
            transition={{
              duration: 4,
              repeat: Infinity,
              ease: "easeInOut",
            }}
          />
          <motion.div
            className="absolute top-1/3 -right-8 w-16 h-16 bg-gradient-to-br from-modern-indigo-200 to-modern-indigo-300 rounded-full opacity-40 blur-sm"
            animate={{
              y: [0, 15, 0],
              x: [0, -5, 0],
            }}
            style={{
              transform: `translateY(${scrollY * -0.15}px)`,
            }}
            transition={{
              duration: 6,
              repeat: Infinity,
              ease: "easeInOut",
              delay: 2,
            }}
          />
        </>
      )}

      <div className="lg:w-6/12 space-y-6 relative z-10">
        <motion.h1
          className="text-4xl md:text-6xl font-display font-bold text-modern-slate-900 leading-tight tracking-tight"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.8, delay: 0.2 }}
        >
          Build{" "}
          <span className="text-modern-emerald-600 relative">
            {hero.titleHighlighted}
            <motion.div
              className="absolute -bottom-1 left-0 w-full h-1 bg-gradient-to-r from-modern-emerald-400 to-modern-emerald-600"
              initial={{ scaleX: 0 }}
              animate={{ scaleX: 1 }}
              transition={{ duration: 0.8, delay: 1 }}
            />
          </span>
          {hero.titleSuffix || ", Not Just Notes"}
        </motion.h1>

        <motion.p
          className="text-xl font-body text-modern-slate-600 leading-relaxed"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.8, delay: 0.4 }}
        >
          {hero.description}
        </motion.p>

        {hero.outcomes && (
          <motion.ul
            className="mt-6 space-y-2"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.8, delay: 0.5 }}
          >
            {hero.outcomes.map((outcome, index) => (
              <motion.li
                key={index}
                className="flex items-start gap-2 text-base font-body text-modern-slate-700"
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.6, delay: 0.6 + index * 0.1 }}
              >
                <span className="text-modern-emerald-600 mt-0.5 flex-shrink-0">✓</span>
                <span>{outcome}</span>
              </motion.li>
            ))}
          </motion.ul>
        )}

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.6 }}
        >
          <motion.button
            onClick={onSignUp}
            whileHover={{
              scale: 1.05,
              y: -2,
              boxShadow:
                "0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)",
            }}
            whileTap={{ scale: 0.98 }}
            className="px-8 py-4 bg-gradient-to-r from-modern-emerald-600 to-modern-emerald-700 text-white rounded-xl font-body font-semibold text-lg hover:from-modern-emerald-700 hover:to-modern-emerald-800 transition-all duration-300 shadow-lg relative overflow-hidden group"
            aria-label={hero.buttonText}
          >
            <span className="relative z-10">{hero.buttonText}</span>
            <motion.div
              className="absolute inset-0 bg-gradient-to-r from-modern-emerald-700 to-modern-emerald-800"
              initial={{ x: "-100%" }}
              whileHover={{ x: "0%" }}
              transition={{ duration: 0.3 }}
            />
          </motion.button>
        </motion.div>
      </div>

      <motion.div
        className="lg:w-6/12 relative"
        initial={{ opacity: 0, scale: 0.8 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.8, delay: 0.4 }}
        whileHover={{
          scale: 1.02,
          rotateY: 2,
          rotateX: 1,
        }}
      >
        <motion.div
          className="absolute -inset-4 bg-gradient-to-r from-modern-emerald-400 to-modern-indigo-400 rounded-xl opacity-20 blur-lg"
          animate={{
            scale: [1, 1.05, 1],
            opacity: [0.2, 0.3, 0.2],
          }}
          transition={{
            duration: 3,
            repeat: Infinity,
            ease: "easeInOut",
          }}
        />
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
    </motion.div>
  );
}
