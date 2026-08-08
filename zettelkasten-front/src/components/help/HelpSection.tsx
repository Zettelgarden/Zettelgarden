import React from 'react';
import { HelpSection as HelpSectionType } from '../../types/help';
import { HelpContent } from './HelpContent';

interface HelpSectionProps {
  section: HelpSectionType;
  className?: string;
}

export function HelpSection({ section, className = '' }: HelpSectionProps) {
  const levelColors = {
    beginner: 'text-green-600',
    intermediate: 'text-yellow-600',
    advanced: 'text-red-600',
  };

  return (
    <section className={`mb-8 ${className}`}>
      <div className="flex items-center gap-3 mb-4">
        <span className="text-2xl">{section.icon}</span>
        <h2 className="text-2xl font-semibold">{section.title}</h2>
        <span
          className={`text-sm px-2 py-1 rounded-full border ${
            levelColors[section.level]
          }`}
        >
          {section.level}
        </span>
      </div>

      <div className="space-y-4">
        {section.content.map((content, index) => (
          <HelpContent key={index} content={content} />
        ))}
      </div>
    </section>
  );
}
