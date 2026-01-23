export interface SectionContent {
  title: string;
  description: string;
}

export interface Feature {
  id: string;
  title: string;
  summary: string;
  details: string;
  icon: string;
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
  description: string;
  buttonText: string;
}

export interface VideoContent {
  youtubeId: string;
  title: string;
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
