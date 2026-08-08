import { HelpSection, HelpCategory } from '../types/help';

export const helpCategories: HelpCategory[] = [
  {
    id: 'quickstart',
    label: 'Quick Start',
    icon: '🚀',
    description: 'Get up and running quickly',
  },
  {
    id: 'features',
    label: 'Features',
    icon: '⚡',
    description: 'Explore what Zettelgarden can do',
  },
  {
    id: 'tutorials',
    label: 'Tutorials',
    icon: '📚',
    description: 'Step-by-step guides',
  },
  {
    id: 'advanced',
    label: 'Advanced',
    icon: '🔧',
    description: 'Power user features',
  },
  {
    id: 'faq',
    label: 'FAQ',
    icon: '❓',
    description: 'Common questions answered',
  },
];

export const helpSections: HelpSection[] = [
  {
    id: 'welcome',
    title: 'Welcome to Zettelgarden',
    icon: '🌱',
    category: 'quickstart',
    level: 'beginner',
    order: 1,
    content: [
      {
        type: 'text',
        data: 'Welcome to your knowledge garden! Zettelgarden is an open-source personal knowledge management system that preserves human insight while leveraging modern technology. Built on zettelkasten principles, it helps you develop and maintain your own understanding of the world.',
      },
      {
        type: 'callout',
        data: {
          type: 'success',
          title: '🌿 Getting Started',
          message:
            'Create your first card by clicking the "+" button, then experiment with different card types and relationships.',
        },
      },
    ],
  },
  {
    id: 'ai-features',
    title: 'AI-Powered Analysis & Summarization',
    icon: '🤖',
    category: 'features',
    level: 'beginner',
    order: 2,
    content: [
      {
        type: 'text',
        data: 'While other tools rush to automate everything with LLMs, Zettelgarden takes a measured approach. AI features are designed to augment your thinking process, not replace it.',
      },
      {
        type: 'text',
        data: 'Transform dense articles, podcasts, or research into clear two-part outputs:',
      },
      {
        type: 'list',
        data: [
          'Executive Summaries: Concise, strategic, and outcome-focused summaries for decision-makers.',
          'Reference Summaries: Detailed, factual, and precise summaries with ranked arguments and verifiable facts for researchers.',
        ],
      },
    ],
  },
  {
    id: 'facts-entities',
    title: 'Facts & Entities - Granular Insights',
    icon: '🔬',
    category: 'features',
    level: 'intermediate',
    order: 3,
    content: [
      {
        type: 'text',
        data: 'Go beyond high-level summaries and explore the building blocks of knowledge:',
      },
      {
        type: 'list',
        data: [
          'Entities: Automatically identify and extract key concepts, people, places, and organizations from your notes to reveal hidden connections.',
          'Facts: Automatically pull out discrete, verifiable statements (like statistics, events, or claims of evidence) from your source material, allowing you to build arguments on a solid foundation.',
        ],
      },
      {
        type: 'callout',
        data: {
          type: 'info',
          title: 'PRO Feature',
          message: 'Facts and Entities are available with a PRO subscription.',
        },
      },
    ],
  },
  {
    id: 'cards',
    title: 'Cards - Your Knowledge Building Blocks',
    icon: '📝',
    category: 'features',
    level: 'beginner',
    order: 4,
    content: [
      {
        type: 'text',
        data: 'Cards are the fundamental building blocks of your knowledge garden. Each card represents a single idea, concept, or piece of information.',
      },
      {
        type: 'list',
        data: [
          'Create new cards for any type of content',
          'Link cards together to create knowledge networks',
          'Create hierarchical relationships between cards',
          'Use markdown formatting for rich content',
        ],
      },
    ],
  },
  {
    id: 'tasks',
    title: 'Tasks - Turn Knowledge Into Action',
    icon: '✅',
    category: 'features',
    level: 'beginner',
    order: 5,
    content: [
      {
        type: 'text',
        data: 'Transform your knowledge into actionable items:',
      },
      {
        type: 'list',
        data: [
          'Create tasks directly from your cards',
          'Track progress on your projects',
          'Set priorities and due dates',
          'Organize tasks into projects and sprints',
        ],
      },
    ],
  },
];
