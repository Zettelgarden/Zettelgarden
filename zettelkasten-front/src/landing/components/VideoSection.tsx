import React from "react";
import { motion } from "framer-motion";
import type { VideoContent } from "../../types/landing";

interface VideoSectionProps {
  video: VideoContent;
}

export function VideoSection({ video }: VideoSectionProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="mt-24 mb-12"
    >
      <h2 className="text-3xl font-display font-bold text-center mb-8 text-modern-slate-900">
        {video.title}
      </h2>
      <div
        className="relative w-full"
        style={{ paddingBottom: "56.25%" }}
      >
        <iframe
          src={`https://www.youtube.com/embed/${video.youtubeId}`}
          title="Zettelgarden Demo"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
          className="absolute top-0 left-0 w-full h-full rounded-xl shadow-2xl"
        ></iframe>
      </div>
    </motion.div>
  );
}
