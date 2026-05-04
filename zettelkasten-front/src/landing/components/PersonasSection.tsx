import React from "react";
import { motion } from "framer-motion";
import type { Persona } from "../../types/landing";

interface PersonasSectionProps {
  personas: Persona[];
}

export function PersonasSection({ personas }: PersonasSectionProps) {
  return (
    <div className="py-12">
      <motion.div
        initial={{ opacity: 0 }}
        whileInView={{ opacity: 1 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6 }}
        className="grid grid-cols-2 lg:grid-cols-4 gap-4"
      >
        {personas.map((persona, index) => {
          const IconComponent = persona.icon;
          return (
            <motion.div
              key={persona.id}
              initial={{ opacity: 0, y: 10 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.4, delay: index * 0.08 }}
              className="flex items-center gap-3 rounded-lg bg-white/60 border border-modern-slate-200/50 px-4 py-3"
            >
              <span className="flex items-center justify-center w-8 h-8 rounded-md bg-modern-emerald-50 text-modern-emerald-600 flex-shrink-0" aria-hidden="true">
                <IconComponent className="w-4 h-4" />
              </span>
              <div className="min-w-0">
                <p className="text-sm font-semibold text-modern-slate-900 leading-tight truncate">
                  {persona.title}
                </p>
                <p className="text-xs text-modern-slate-500 leading-snug line-clamp-2">
                  {persona.description}
                </p>
              </div>
            </motion.div>
          );
        })}
      </motion.div>
    </div>
  );
}
