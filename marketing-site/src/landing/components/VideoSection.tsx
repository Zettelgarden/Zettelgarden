import React from "react";
import { motion } from "framer-motion";
import type { VideoContent } from "../../types/landing";

interface VideoSectionProps {
  video: VideoContent;
  onCtaClick?: () => void;
}

export function VideoSection({ video, onCtaClick }: VideoSectionProps) {
  return (
    <motion.div
      id="video"
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="py-24"
    >
      <h2 className="text-3xl font-display font-bold text-center mb-8 text-modern-slate-900">
        {video.title}
      </h2>
      <div
        className="relative w-full"
        style={{ paddingBottom: "56.25%" }}
      >
        <iframe
          src={`https://www.youtube-nocookie.com/embed/${video.youtubeId}`}
          title="Zettelgarden Demo"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
          className="absolute top-0 left-0 w-full h-full rounded-xl shadow-2xl"
        ></iframe>
      </div>

      {video.ctaText && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="mt-12 text-center"
        >
          <h3 className="text-2xl font-display font-semibold text-modern-slate-900 mb-2">
            {video.ctaText}
          </h3>
          {video.ctaSubtext && (
            <p className="text-modern-slate-600 mb-6">{video.ctaSubtext}</p>
          )}
          {onCtaClick && (
            <motion.button
              onClick={onCtaClick}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.98 }}
              className="px-8 py-3 bg-modern-emerald-600 text-white rounded-lg font-body font-semibold hover:bg-modern-emerald-700 transition-colors duration-200 shadow-lg"
            >
              Get Started Free
            </motion.button>
          )}
        </motion.div>
      )}
    </motion.div>
  );
}
