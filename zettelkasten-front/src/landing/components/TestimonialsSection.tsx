import React from "react";
import { motion } from "framer-motion";
import type { Testimonial } from "../../types/landing";

interface TestimonialsSectionProps {
  testimonials: Testimonial[];
  sectionTitle: string;
  sectionDescription: string;
}

function TestimonialCard({ testimonial, index }: { testimonial: Testimonial; index: number }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6, delay: index * 0.1 }}
      whileHover={{ y: -4 }}
      className="bg-white/70 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300"
    >
      <div className="flex gap-1 mb-4">
        {[...Array(5)].map((_, i) => (
          <svg
            key={i}
            className="w-5 h-5 text-modern-emerald-500 fill-current"
            viewBox="0 0 20 20"
          >
            <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
          </svg>
        ))}
      </div>
      <blockquote className="font-body text-modern-slate-700 leading-relaxed mb-6">
        "{testimonial.quote}"
      </blockquote>
      <div className="flex items-center gap-3">
        {testimonial.avatar ? (
          <img
            src={testimonial.avatar}
            alt={testimonial.author}
            className="w-12 h-12 rounded-full object-cover"
          />
        ) : (
          <div className="w-12 h-12 rounded-full bg-gradient-to-br from-modern-emerald-400 to-modern-emerald-600 flex items-center justify-center text-white font-semibold text-lg">
            {testimonial.author.charAt(0)}
          </div>
        )}
        <div>
          <p className="font-display font-semibold text-modern-slate-900">
            {testimonial.author}
          </p>
          {testimonial.role && (
            <p className="font-body text-sm text-modern-slate-600">
              {testimonial.role}
            </p>
          )}
        </div>
      </div>
    </motion.div>
  );
}

export function TestimonialsSection({
  testimonials,
  sectionTitle,
  sectionDescription,
}: TestimonialsSectionProps) {
  return (
    <div className="py-24 space-y-12">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6 }}
        className="text-center"
      >
        <h2 className="text-4xl font-display font-bold text-modern-slate-900 mb-4">
          {sectionTitle}
        </h2>
        <p className="text-xl font-body text-modern-slate-600 max-w-3xl mx-auto">
          {sectionDescription}
        </p>
      </motion.div>

      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
        {testimonials.map((testimonial, index) => (
          <TestimonialCard
            key={testimonial.id}
            testimonial={testimonial}
            index={index}
          />
        ))}
      </div>
    </div>
  );
}
