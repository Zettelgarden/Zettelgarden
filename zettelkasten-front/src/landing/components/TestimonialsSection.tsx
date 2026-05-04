import React from "react";
import { motion } from "framer-motion";
import { Github, Star, GitFork, BookOpen } from "lucide-react";

interface TestimonialsSectionProps {
  testimonials: never[];
  sectionTitle: string;
  sectionDescription: string;
}

export function TestimonialsSection({
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

      <div className="grid md:grid-cols-3 gap-6 max-w-4xl mx-auto">
        <motion.a
          href="https://github.com/NickSavage/Zettelgarden"
          target="_blank"
          rel="noopener noreferrer"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0 }}
          whileHover={{ y: -4 }}
          className="bg-white/70 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300 text-center"
        >
          <div className="flex items-center justify-center w-12 h-12 rounded-full bg-modern-slate-900 text-white mx-auto mb-4">
            <Github className="w-6 h-6" />
          </div>
          <h3 className="font-display font-semibold text-modern-slate-900 mb-2">
            View Source Code
          </h3>
          <p className="font-body text-sm text-modern-slate-600">
            Full source code available on GitHub. Inspect, modify, and contribute.
          </p>
        </motion.a>

        <motion.a
          href="https://github.com/NickSavage/Zettelgarden/stargazers"
          target="_blank"
          rel="noopener noreferrer"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.1 }}
          whileHover={{ y: -4 }}
          className="bg-white/70 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300 text-center"
        >
          <div className="flex items-center justify-center w-12 h-12 rounded-full bg-modern-emerald-600 text-white mx-auto mb-4">
            <Star className="w-6 h-6" />
          </div>
          <h3 className="font-display font-semibold text-modern-slate-900 mb-2">
            Star on GitHub
          </h3>
          <p className="font-body text-sm text-modern-slate-600">
            Support the project and stay updated on new releases.
          </p>
        </motion.a>

        <motion.a
          href="https://github.com/NickSavage/Zettelgarden/blob/master/CONTRIBUTING.md"
          target="_blank"
          rel="noopener noreferrer"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          whileHover={{ y: -4 }}
          className="bg-white/70 backdrop-blur-sm rounded-xl p-6 border border-modern-slate-200/50 hover:border-modern-emerald-300/50 transition-all duration-300 text-center"
        >
          <div className="flex items-center justify-center w-12 h-12 rounded-full bg-modern-indigo-600 text-white mx-auto mb-4">
            <GitFork className="w-6 h-6" />
          </div>
          <h3 className="font-display font-semibold text-modern-slate-900 mb-2">
            Contribute
          </h3>
          <p className="font-body text-sm text-modern-slate-600">
            Help build the future of knowledge management. PRs welcome.
          </p>
        </motion.a>
      </div>
    </div>
  );
}
