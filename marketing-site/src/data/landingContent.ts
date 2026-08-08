import {
  Brain,
  Sprout,
  Link2,
  Bot,
  ClipboardList,
  Unlock,
  Rss,
  CheckSquare,
  FolderOpen,
} from "lucide-react";
import type {
  Feature,
  PricingTier,
  HeroContent,
  VideoContent,
  NewsletterContent,
  PricingFeature,
  SectionContent,
  Persona,
  FAQ,
  BuiltByContent,
} from "../types/landing";

// Shared feature arrays for pricing tiers
const freeFeatures: PricingFeature[] = [
  { text: "Atomic Notes & Cards" },
  { text: "Bidirectional Linking" },
  { text: "Task Management" },
  { text: "RSS Reader" },
  { text: "File Uploads & Management" },
  { text: "Basic Search" },
];

const proFeatures: PricingFeature[] = [
  { text: "Everything in Free" },
  { text: "AI Chat with Knowledge Base", highlight: true },
  { text: "Vector/Semantic Search", highlight: true },
  { text: "Entity Recognition & Linking", highlight: true },
  { text: "Content Analysis & Summaries", highlight: true },
  { text: "Custom AI Agents", highlight: true },
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
    title: "AI That Augments Your Thinking",
    summary:
      "Discover connections and patterns you'd miss manually—without losing control to AI.",
    details:
      "Unlike tools that replace your thinking with automated summaries, Zettelgarden helps you develop genuine insights. AI acts as a research assistant that surfaces relevant connections and patterns in your knowledge, while you remain the decision-maker. Every connection you make strengthens your personal understanding.",
    icon: Brain,
  },
  {
    id: "zettelkasten-method",
    title: "A System for Breakthrough Insights",
    summary:
      "The same method used by history's most productive thinkers—now enhanced with AI.",
    details:
      "Darwin, Luhmann, and countless others used the zettelkasten method to produce breakthrough work. Instead of filing notes in folders, you link ideas together like your brain does—creating a web of knowledge that grows smarter over time. This isn't just note-taking; it's a thinking methodology that compounds.",
    icon: Sprout,
  },
  {
    id: "connected-knowledge",
    title: "See Ideas Connect Automatically",
    summary:
      "Link any note to any other and watch connections emerge across your entire knowledge base.",
    details:
      "When you connect Card A to Card B, the link works both ways—automatically. Over time, you'll see unexpected connections: that research note from three years ago relates to today's project. Your knowledge compounds instead of collecting dust, with visual representations showing how ideas interrelate.",
    icon: Link2,
  },
  {
    id: "ai-chat",
    title: "Ask Your Knowledge Base Anything",
    summary:
      "Search, analyze, and synthesize information across all your notes with an AI research assistant.",
    details:
      "Our AI doesn't just chat—it actively works with your knowledge. Ask 'What did I learn about topic X?' and it searches through every card, finds connections, creates new notes if needed, and provides insights by combining information from multiple sources. It's like having a research assistant who has read everything you've written.",
    icon: Bot,
  },
  {
    id: "summaries",
    title: "Turn Articles into Actionable Insights",
    summary:
      "Paste any article, paper, or transcript—get a structured summary with theses, arguments, and facts.",
    details:
      "Save hours of reading time. Drop in a dense research paper, podcast transcript, or long article and get a clear summary with the main thesis, ranked arguments, and verifiable facts extracted for you. Each summary preserves the original context while making information immediately usable for your work.",
    icon: ClipboardList,
  },
  {
    id: "rss-reader",
    title: "RSS Reader Built for Knowledge Workers",
    summary:
      "Follow your favorite sources, read articles, and convert them into linked notes in one click.",
    details:
      "Stay on top of your information diet without context-switching. Zettelgarden's built-in RSS reader lets you subscribe to blogs, journals, and news sources. Read articles inline, tag and organize them into folders, then convert anything worth remembering into a card with a single click. Your reading becomes part of your knowledge graph.",
    icon: Rss,
  },
  {
    id: "task-management",
    title: "Tasks in One Place",
    summary:
      "Manage your to-dos and keep your action items connected to your notes.",
    details:
      "Your tasks live alongside your knowledge—not in a separate app. Schedule tasks with due dates and link action items to the notes and research that inform them. When you complete a project, the context is already there.",
    icon: CheckSquare,
  },
  {
    id: "files",
    title: "Files & Documents, Organized",
    summary:
      "Upload PDFs, images, and documents. Tag them, search their contents, and link them to your notes.",
    details:
      "Drop files into Zettelgarden and they become first-class citizens in your knowledge base. Upload PDFs, images, epubs, and more—then tag them, search their contents, and connect them to relevant cards. No more digging through folders to find that one document you saved months ago.",
    icon: FolderOpen,
  },
  {
    id: "open-source",
    title: "You Own Your Knowledge—Always",
    summary:
      "Self-host for complete privacy, export anytime, or use our secure cloud. No vendor lock-in.",
    details:
      "Your knowledge belongs to you, not a platform. Export your entire database in standard formats anytime. Self-host for complete data privacy, or trust our secure cloud. Full source code is available on GitHub—you can inspect, modify, and run it yourself. Join our community of contributors building the future of knowledge management together. No data mining, no walled gardens.",
    icon: Unlock,
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
    subtitle: "30-day free trial — Try all AI features",
    features: proFeatures,
    buttonText: "Choose Monthly",
    buttonColor: "indigo",
    route: "/subscribe",
  },
  {
    id: "annual",
    name: "PRO Annual",
    price: "$100 / year (Save 20%)",
    subtitle: "30-day free trial — Try all AI features",
    features: proFeatures,
    buttonText: "Choose Annual",
    buttonColor: "indigo",
    route: "/subscribe",
    recommended: true,
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
  ctaSubtext: "Get started with Zettelgarden Free — no credit card required.",
};

export const pricingSection: SectionContent = {
  title: "Simple, Transparent Pricing",
  description:
    "Start free and discover how AI can augment your thinking. Upgrade to unlock advanced AI agents, content analysis, and discovery features. 30-day free trial included.",
};

export const faqs: FAQ[] = [
  {
    id: "different-from-others",
    question: "How is this different from Obsidian, Notion, or Roam?",
    answer: "Zettelgarden is built around AI-assisted knowledge discovery from the start. Unlike other tools where AI is an add-on, our AI agents can search, analyze, and synthesize across your entire knowledge base automatically. It also includes a built-in RSS reader and file management—so you don't need several separate apps. Plus, we're fully open-source and self-hostable.",
  },
  {
    id: "free-vs-pro",
    question: "What's the difference between Free and PRO?",
    answer: "Free gives you the full knowledge management experience: atomic notes, bidirectional linking, task management, RSS reader, file uploads, and basic search. PRO unlocks AI features: chat with your knowledge base, semantic search that finds related ideas even without exact keywords, entity recognition, content analysis, custom AI agents, and automated summaries.",
  },
  {
    id: "privacy",
    question: "Is my data private? Can I self-host?",
    answer: "Absolutely. Your data is encrypted and never sold. You can export everything in standard formats anytime. For complete privacy, self-host Zettelgarden on your own server—the full source code is available on GitHub.",
  },
  {
    id: "offline",
    question: "Does it work offline?",
    answer: "Zettelgarden requires an internet connection to sync your notes and use AI features. Your data is stored securely in the cloud, with the option to self-host for complete control.",
  },
  {
    id: "getting-started",
    question: "How do I get started?",
    answer: "Just click 'Get Started Free' and create your account. No credit card required. You can create your first note immediately, and we'll show you how to link ideas as you go. Most users are comfortable within 10 minutes.",
  },
  {
    id: "rss-reader",
    question: "Tell me more about the RSS reader.",
    answer: "The RSS reader lets you subscribe to any blog, journal, or feed. Read articles inline, organize them into folders, and convert anything into a linked note with one click. It lives alongside your notes so your knowledge and reading stay connected.",
  },
];

export const faqSection: SectionContent = {
  title: "Frequently Asked Questions",
  description: "Got questions? We've got answers. If you don't see what you're looking for, feel free to reach out.",
};

export const builtByContent: BuiltByContent = {
  founderName: "Nick Savage",
  tagline: ", and I built Zettelgarden to help people turn information overload into understanding.",
  story: "I believe AI should augment human thinking, not replace it. Based in Ottawa, Canada, I'm committed to building tools that help you discover insights in your knowledge that you'd miss manually—while keeping you in control.",
  githubUrl: "https://github.com/NickSavage/Zettelgarden",
};

// Testimonials replaced with open-source community section
export const testimonials: SectionContent[] = [];

export const testimonialsSection: SectionContent = {
  title: "Open Source & Community Driven",
  description:
    "Zettelgarden is open source on GitHub. Join our community of contributors building the future of knowledge management.",
};

export const personas: Persona[] = [
  {
    id: "researchers",
    title: "Academic Researchers",
    description:
      "Manage hundreds of papers and find connections across disciplines. Never lose track of a source or insight again.",
    icon: Brain,
  },
  {
    id: "writers",
    title: "Writers & Authors",
    description:
      "Never lose a research thread or brilliant idea. Connect thoughts across projects and watch your work evolve.",
    icon: ClipboardList,
  },
  {
    id: "students",
    title: "Students & Learners",
    description:
      "Turn lecture notes into long-term understanding. Build a knowledge base that grows with you throughout your education.",
    icon: Sprout,
  },
  {
    id: "professionals",
    title: "Knowledge Workers",
    description:
      "Break information silos and see the big picture. Connect insights from meetings, documents, and research in one place.",
    icon: FolderOpen,
  },
];

export const personasSection: SectionContent = {
  title: "Who is Zettelgarden For?",
  description:
    "Zettelgarden is designed for anyone who wants to think better and remember more. Here's how it helps different people achieve their goals.",
};
