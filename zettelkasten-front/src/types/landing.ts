export interface SectionContent {
  title: string;
  description: string;
  ctaText?: string;
  ctaSubtext?: string;
}

import type { LucideIcon } from "lucide-react";

export interface Feature {
  id: string;
  title: string;
  summary: string;
  details: string;
  icon: LucideIcon;
}

export interface Persona {
  id: string;
  title: string;
  description: string;
  icon: LucideIcon;
}

export interface FAQ {
  id: string;
  question: string;
  answer: string;
}

export interface BuiltByContent {
  founderName: string;
  tagline: string;
  story?: string;
  founderPhoto?: string;
  githubUrl?: string;
}

export interface Testimonial {
  id: string;
  quote: string;
  author: string;
  role?: string;
  avatar?: string;
}

export interface PricingFeature {
  text: string;
  highlight?: boolean;
}

export interface PricingTier {
  id: 'free' | 'monthly' | 'annual';
  name: string;
  price: string;
  subtitle?: string;
  features: PricingFeature[];
  buttonText: string;
  buttonColor: 'green' | 'indigo';
  route: string;
}

export interface HeroContent {
  title: string;
  titleHighlighted: string;
  titleSuffix?: string;
  description: string;
  buttonText: string;
  outcomes?: string[];
}

export interface VideoContent {
  youtubeId: string;
  title: string;
  ctaText?: string;
  ctaSubtext?: string;
}

export interface NewsletterContent {
  title: string;
  description: string;
  inputPlaceholder: string;
  buttonText: string;
  buttonLoadingText: string;
  successMessage: string;
  invalidEmailMessage: string;
}
