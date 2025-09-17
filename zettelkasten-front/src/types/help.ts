export interface HelpContent {
  type: 'text' | 'list' | 'code' | 'video' | 'interactive' | 'callout';
  data: any;
}

export interface HelpSection {
  id: string;
  title: string;
  icon: string;
  category: 'quickstart' | 'features' | 'tutorials' | 'advanced' | 'faq';
  level: 'beginner' | 'intermediate' | 'advanced';
  content: HelpContent[];
  order: number;
}

export type HelpCategory = {
  id: 'quickstart' | 'features' | 'tutorials' | 'advanced' | 'faq';
  label: string;
  icon: string;
  description: string;
};