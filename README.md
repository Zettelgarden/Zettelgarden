# Zettelgarden

**Build Understanding, Not Just Notes**

Zettelgarden is the knowledge management system that thinks with you. Combining proven zettelkasten methodology with AI intelligence, it helps knowledge workers, researchers, and thinkers discover connections, build insights, and turn information overload into understanding.

**Target Users**: Researchers, consultants, content creators, students, and anyone who works with complex information and wants to build connected understanding rather than just store data.

**NOTE: This project is actively evolving. While stable for personal use, features may change based on community feedback and development priorities.**

## Why Zettelgarden?

**Unique Positioning**: The only knowledge management platform that combines proven zettelkasten methodology with built-in AI features, offering web accessibility with self-hosting options.

### vs. Competitors
- **vs. Obsidian**: Web-native (no desktop app required), built-in AI features, mobile-optimized
- **vs. Notion**: Purpose-built for knowledge connection, faster search, academic methodology
- **vs. Roam**: Modern interface, AI integration, better performance, open source
- **vs. AI-only tools**: AI augments proven methodology instead of replacing it

### Core Principles
- **Human-Centric AI**: AI augments your thinking rather than replacing it
- **Proven Methodology**: Based on the zettelkasten system used by Darwin, Luhmann, and other great thinkers
- **Connected Knowledge**: Every idea links to every other idea in a living knowledge graph
- **Privacy First**: Self-host for complete control or use secure cloud hosting

## Demo

Watch our [demo video](https://www.youtube.com/watch?v=0kSAhX2R7eM) to see Zettelgarden in action and learn about its key features.

You can also try Zettelgarden directly at [zettelgarden.com](https://zettelgarden.com) using our demo account:
- Email: demo@zettelgarden.com
- Password: demo

## Features

### Core Knowledge Management
- **Atomic Cards**: Markdown-supported notes with unique identifiers for reliable linking
- **Bidirectional Linking**: Automatic `[[card-title]]` syntax with backlink detection and display
- **Task Management**: Integrated todo system with scheduling, priorities, and recurring patterns
- **File Attachments**: Upload and organize PDFs, images, and documents with local on-disk storage
- **Templates**: Reusable card templates with variable substitution
- **Hierarchical Organization**: Parent-child card relationships with multiple view modes

### AI-Powered Intelligence (PRO Features)
- **Vector Search**: Semantic similarity search using embeddings for content discovery beyond keywords
- **Entity Recognition**: Automatic extraction and linking of people, places, organizations, and concepts using LLM-powered NLP
- **Content Analysis**: AI-generated summaries, theme extraction, and insight generation with citation integrity
- **Smart Discovery**: Related content recommendations and pattern recognition across your knowledge graph

### Search & Discovery
- **Multi-Modal Search**:
  - Full-text search with Typesense integration
  - Vector/semantic search using OpenAI-compatible embeddings
  - Boolean operators and advanced filters
  - Real-time search results with highlighting
- **Entity-Based Navigation**: Search and filter by automatically recognized entities
- **Starred Searches**: Save and organize frequently used search queries
- **Backlink Analysis**: Automatic bidirectional relationship tracking

### Technical Features
- **Self-Hosting**: Docker-based deployment with SQLite — no external database required — for complete data ownership
- **Web-Native**: Full functionality in browser with Progressive Web App capabilities
- **API Access**: RESTful API with JWT authentication for programmatic integration
- **Real-Time Sync**: WebSocket connections with optimistic updates across sessions
- **Data Portability**: One-click full-data export from Settings — a zip of every user-owned table (JSON), cards as Markdown/CSV, and your original uploaded files. Migration tooling lives under `docs/migration-plans/`.
- **Keyboard Shortcuts**: Efficient power-user workflows (c=create, s=search, t=tasks)

### Privacy & Security
- **Data Ownership**: Complete control with self-hosting option or secure cloud hosting
- **Encryption**: TLS for transport, AES for storage
- **No Vendor Lock-in**: Export your data anytime and delete your account self-serve from Settings (admins can also delete any user), open source transparency
- **Privacy-First AI**: Optional AI features, no data mining, model choices

## Architecture

Built with modern, scalable technologies optimized for both performance and AI capabilities:

### Frontend (`zettelkasten-front`)
- **Framework**: React 18 with TypeScript for type safety
- **Build Tool**: Vite for fast development and optimized builds
- **Styling**: Tailwind CSS with custom design system
- **State Management**: React Context API with optimistic updates
- **Real-time**: WebSocket integration for live synchronization
- **PWA**: Service worker with offline capabilities and caching

### Backend (`go-backend`)
- **Language**: Go with `net/http` for high-performance HTTP server
- **Database**: SQLite (file-based, WAL mode) for storage — no external database server
- **Search Engine**: Typesense for full-text search with built-in ML capabilities
- **Authentication**: JWT-based with middleware pipeline
- **File Storage**: Local on-disk storage under STORAGE_DIR (no external object store)
- **API Design**: RESTful endpoints with JSON responses

### AI/ML Stack
- **Embeddings**: OpenAI-compatible API integration for text embeddings
- **Vector Search**: Embedding-based cosine similarity for semantic search
- **Entity Recognition**: LLM-powered Named Entity Recognition pipeline
- **Content Analysis**: Structured prompting for summaries and insights
- **Search Integration**: Hybrid search combining full-text (Typesense) and vector similarity
- **Model Flexibility**: Configurable LLM endpoints (OpenAI, Anthropic, local models)

### Infrastructure
- **Containerization**: Docker and Docker Compose for easy deployment
- **Database Migrations**: SQL migration system with version control
- **Monitoring**: Structured logging with configurable levels
- **Security**: TLS/SSL, input validation, SQL injection protection
- **Backup**: Online SQLite snapshots via `VACUUM INTO` (see the [backup runbook](docs/runbooks/sqlite-backup.md))

### Mail Service (`python-mail`)
- **Framework**: Flask for lightweight SMTP service
- **Purpose**: User notifications, password resets, subscription management
- **Integration**: RESTful API for backend communication

## Getting Started

### Quick Start Options

1. **Try the Demo**: Visit [zettelgarden.com](https://zettelgarden.com) and use demo credentials (demo@zettelgarden.com / demo)
2. **Cloud Hosting**: Sign up for free at [zettelgarden.com](https://zettelgarden.com) with 30-day PRO trial
3. **Self-Hosting**: Deploy with Docker for complete data ownership

### Self-Hosting Requirements
- Docker and Docker Compose
- SQLite database (bundled — no external database server to install)
- Typesense search server
- Local disk for file storage (bundled — no external object store)
- OpenAI-compatible API key for AI features (optional)
- OIDC / SSO provider for single sign-on (optional, e.g. [Pocket ID](https://github.com/pocket-id/pocket-id)) — see `go-backend/.env.example` (`OIDC_*` vars) and `docs/plans/2026-08-03-oidc-authentication-design.md`

See our [getting started guide](https://zettelgarden.com/docs/getting-started/) for detailed setup instructions. (Coming soon!)

### Use Cases

**For Researchers**: Organize literature, track citations, discover cross-study connections
**For Consultants**: Build client knowledge bases, synthesize industry insights, develop frameworks
**For Content Creators**: Research management, idea development, content planning workflows
**For Students**: Note-taking that builds understanding, concept mapping, exam preparation

## Pricing

- **Free**: Core zettelkasten features (cards, linking, tasks, basic search)
- **PRO ($10/month, $100/year)**: AI features, vector search, entity recognition, content analysis
- **Self-Hosting**: Deploy the open source version with your own infrastructure

## Contributing

Zettelgarden is built in the open. Contributions and feedback are welcome. Please check our [contribution guidelines](CONTRIBUTING.md).

## Stay Updated

Follow our [blog](https://zettelgarden.com/blog/) and Nick Savage's [Substack](https://nsavage.substack.com/) for detailed updates on development and new features.