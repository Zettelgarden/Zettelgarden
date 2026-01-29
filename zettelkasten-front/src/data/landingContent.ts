import type {
  Feature,
  PricingTier,
  HeroContent,
  VideoContent,
  NewsletterContent,
  PricingFeature,
  SectionContent,
  Persona,
} from "../types/landing";

// Shared feature arrays for pricing tiers (eliminates duplication)
const freeFeatures: PricingFeature[] = [
  { text: "Atomic Notes & Cards" },
  { text: "Bidirectional Linking" },
  { text: "Task Management" },
  { text: "Basic Search" },
];

const proFeatures: PricingFeature[] = [
  { text: "Everything in Free" },
  { text: "AI Chat with Knowledge Base", highlight: true },
  { text: "Vector/Semantic Search", highlight: true },
  { text: "Entity Recognition & Linking", highlight: true },
  { text: "Content Analysis & Summaries", highlight: true },
  { text: "Early Access to New Features" },
];

export const heroSection: HeroContent = {
  title: "Build",
  titleHighlighted: "Understanding",
  titleSuffix: ", Not Just Notes",
  description:
    "The knowledge management system that thinks with you. Zettelgarden combines proven zettelkasten methodology with AI intelligence to help you discover connections, build insights, and turn information overload into understanding.",
  buttonText: "Get Started with Zettelgarden",
  outcomes: [
    "Never lose a brilliant idea again",
    "Find connections you'd miss manually",
    "Turn information overload into understanding",
  ],
};

export const features: Feature[] = [
  {
    id: "human-centric",
    title: "Human-Centric AI",
    summary:
      "AI augments your thinking rather than replacing it. See relationships between ideas you never noticed before.",
    details:
      "Built on time-tested zettelkasten principles, Zettelgarden helps you develop genuine insights rather than just collecting automated summaries. Every connection you make strengthens your personal knowledge graph, with AI helping you discover patterns you'd miss manually.",
    icon: "🧠",
  },
  {
    id: "zettelkasten-method",
    title: "Proven Zettelkasten Method",
    summary:
      "Based on the system used by history's most productive thinkers like Darwin and Luhmann.",
    details:
      "Atomic notes with bidirectional linking create a knowledge network that grows smarter over time. This isn't just note-taking—it's a thinking methodology that has powered breakthrough insights for centuries, now enhanced with modern technology.",
    icon: "🌱",
  },
  {
    id: "connected-knowledge",
    title: "Connected Knowledge Graph",
    summary:
      "Every idea links to every other idea. Turn information silos into a living knowledge network.",
    details:
      "Bidirectional linking reveals unexpected connections across time and topics. Your knowledge compounds instead of collecting dust, with visual representations showing how your understanding connects and evolves.",
    icon: "🔗",
  },
  {
    id: "ai-chat",
    title: "AI Agents for Discovery",
    summary:
      "Intelligent AI agents that can search, analyze, and synthesize information from your personal knowledge collection.",
    details:
      "Our AI agents don't just chat—they actively work with your knowledge base using sophisticated tools. They can search through your cards, create new notes, analyze patterns, and provide insights by combining information from multiple sources. These agents understand context and can perform complex reasoning tasks across your entire knowledge graph.",
    icon: "🤖",
  },
  {
    id: "summaries",
    title: "Structured Analysis",
    summary:
      "Transform dense articles, podcasts, or research into clear, actionable insights.",
    details:
      "Concise executive summaries for decision‑makers and detailed reference summaries with theses, ranked arguments, and verifiable facts for researchers. Each summary preserves the original context while making information actionable.",
    icon: "📋",
  },
  {
    id: "open-source",
    title: "Your Knowledge, Your Control",
    summary:
      "Self-host for complete privacy or use our secure cloud. No vendor lock-in, no data mining.",
    details:
      "Your knowledge belongs to you—export your data anytime, self-host for complete control, or trust our secure cloud infrastructure. Full source code is available on GitHub with comprehensive documentation.",
    icon: "🔓",
  },
];

export const pricingTiers: PricingTier[] = [
  {
    id: "free",
    name: "Free",
    price: "$0 / forever",
    features: freeFeatures,
    buttonText: "Get Started Free",
    buttonColor: "green",
    route: "/app",
  },
  {
    id: "monthly",
    name: "PRO Monthly",
    price: "$10 / month",
    subtitle: "🎯 30-day free trial - Try all AI features",
    features: proFeatures,
    buttonText: "Choose Monthly",
    buttonColor: "indigo",
    route: "/subscribe",
  },
  {
    id: "annual",
    name: "PRO Annual",
    price: "$100 / year (Save 20%)",
    subtitle: "🎯 30-day free trial - Try all AI features",
    features: proFeatures,
    buttonText: "Choose Annual",
    buttonColor: "indigo",
    route: "/subscribe",
  },
];

export const videoSection: VideoContent = {
  youtubeId: "0kSAhX2R7eM",
  title: "See Zettelgarden in Action",
  ctaText: "See It in Action Yourself",
  ctaSubtext: "Get started in 30 seconds with Zettelgarden Free.",
};

export const newsletterSection: NewsletterContent = {
  title: "Stay Updated",
  description:
    "Stay updated with Zettelgarden's development. Sign up for occasional updates about new features and releases.",
  inputPlaceholder: "Enter your email",
  buttonText: "Sign Up",
  buttonLoadingText: "Signing up...",
  successMessage: "Thank you for signing up!",
  invalidEmailMessage: "Please enter a valid email address",
};

export const featuresSection: SectionContent = {
  title: "Features that Work for You",
  description:
    "Click on any feature to learn more about how Zettelgarden can enhance your knowledge management workflow.",
  ctaText: "Ready to Start Connecting Ideas?",
  ctaSubtext: "Get started with Zettelgarden Free - no credit card required.",
};

export const pricingSection: SectionContent = {
  title: "Simple, Transparent Pricing",
  description:
    "Start free and discover how AI can augment your thinking. Upgrade to unlock advanced AI agents, content analysis, and discovery features. 30-day free trial included.",
};

export const personas: Persona[] = [
  {
    id: "researchers",
    title: "Academic Researchers",
    description:
      "Manage hundreds of papers and find connections across disciplines. Never lose track of a source or insight again.",
    icon: "🎓",
  },
  {
    id: "writers",
    title: "Writers & Authors",
    description:
      "Never lose a research thread or brilliant idea. Connect thoughts across projects and watch your work evolve.",
    icon: "✍️",
  },
  {
    id: "students",
    title: "Students & Learners",
    description:
      "Turn lecture notes into long-term understanding. Build a knowledge base that grows with you throughout your education.",
    icon: "📚",
  },
  {
    id: "professionals",
    title: "Knowledge Workers",
    description:
      "Break information silos and see the big picture. Connect insights from meetings, documents, and research in one place.",
    icon: "💼",
  },
];

export const personasSection: SectionContent = {
  title: "Who is Zettelgarden For?",
  description:
    "Zettelgarden is designed for anyone who wants to think better and remember more. Here's how it helps different people achieve their goals.",
};
