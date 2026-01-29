import React from "react";
import { motion } from "framer-motion";
import type { Persona } from "../../types/landing";

interface PersonasSectionProps {
  personas: Persona[];
  sectionTitle: string;
  sectionDescription: string;
}

function PersonaCard({ persona }: { persona: Persona }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      whileHover={{ y: -4 }}
      className="bg-white/50 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300"
    >
      <div className="flex items-center gap-3 mb-4">
        <span className="text-3xl" aria-hidden="true">
          {persona.icon}
        </span>
        <h3 className="text-xl font-display font-semibold text-modern-slate-900">
          {persona.title}
        </h3>
      </div>
      <p className="font-body text-modern-slate-700 leading-relaxed">
        {persona.description}
      </p>
    </motion.div>
  );
}

export function PersonasSection({
  personas,
  sectionTitle,
  sectionDescription,
}: PersonasSectionProps) {
  return (
    <div className="py-24 space-y-8">
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
        transition={{ duration: 0.8 }}
        className="grid md:grid-cols-2 lg:grid-cols-4 gap-6"
      >
        {personas.map((persona, index) => (
          <motion.div
            key={persona.id}
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6, delay: index * 0.1 }}
          >
            <PersonaCard persona={persona} />
          </motion.div>
        ))}
      </motion.div>
    </div>
  );
}
