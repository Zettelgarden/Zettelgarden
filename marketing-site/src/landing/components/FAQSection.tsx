import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ChevronDownIcon } from "../../assets/icons/ChevronDownIcon";
import type { FAQ } from "../../types/landing";

interface FAQSectionProps {
  faqs: FAQ[];
  sectionTitle: string;
  sectionDescription: string;
}

function FAQItem({ faq }: { faq: FAQ }) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      className="border border-modern-slate-200 rounded-xl overflow-hidden bg-white/50 backdrop-blur-sm"
    >
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full px-6 py-4 flex items-center justify-between text-left hover:bg-modern-slate-50/50 transition-colors duration-200"
        aria-expanded={isOpen}
      >
        <span className="font-body font-medium text-modern-slate-900 pr-4">
          {faq.question}
        </span>
        <motion.div
          animate={{ rotate: isOpen ? 180 : 0 }}
          transition={{ duration: 0.2 }}
          className="flex-shrink-0"
        >
          <ChevronDownIcon className="w-5 h-5 text-modern-slate-600" />
        </motion.div>
      </button>
      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="overflow-hidden"
          >
            <div className="px-6 pb-4 pt-0">
              <p className="font-body text-modern-slate-700 leading-relaxed">
                {faq.answer}
              </p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}

export function FAQSection({
  faqs,
  sectionTitle,
  sectionDescription,
}: FAQSectionProps) {
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

      <div className="max-w-3xl mx-auto space-y-4">
        {faqs.map((faq, index) => (
          <motion.div
            key={faq.id}
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: index * 0.05 }}
          >
            <FAQItem faq={faq} />
          </motion.div>
        ))}
      </div>
    </div>
  );
}
