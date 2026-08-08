> **ARCHIVED** — Historical document moved to `docs/archive/` on 2026-08-08 during the documentation audit (Zettelgarden-0ui). Does not describe the current app; kept for the record.

# Zettelgarden Feature Specifications

## Core Features

### Cards (Atomic Notes)
**Description**: The fundamental unit of knowledge in Zettelgarden
**Technical Implementation**: React components with markdown rendering, stored in PostgreSQL
**User Experience**:
- Create with 'c' keyboard shortcut
- Markdown editor with live preview
- Unique ID generation for reliable linking
- Multiple view modes (normal, summary, analysis)
- Parent-child hierarchical relationships

**Features**:
- Markdown content with syntax highlighting
- Bidirectional linking with `[[card-title]]` syntax
- Star/bookmark functionality
- Card metadata (creation date, last modified, word count)
- File attachments and media embeds
- Tabbed interface for files, facts, and metadata
- Export capabilities

**PRO Features**:
- AI-generated summaries and analysis
- Entity extraction and linking
- Content insights and patterns

### Linking System
**Description**: Connect ideas across your knowledge base
**Technical Implementation**: PostgreSQL relationships with automatic bidirectional updates
**User Experience**:
- Type `[[` to trigger card search autocomplete
- Automatic backlink detection and display
- Visual link indicators and previews
- Orphaned card detection

**Features**:
- Bidirectional linking
- Link previews on hover
- Backlink sections in each card
- Link graph visualization
- Broken link detection

### Search System
**Description**: Multi-modal search across all content
**Technical Implementation**: Typesense for full-text, pgvector for semantic search
**User Experience**:
- Global search with 's' keyboard shortcut
- Real-time search results
- Search within cards
- Starred search queries

**Features**:
- Full-text search across cards and files
- Boolean operators and filters
- Search result highlighting
- Search history and starring
- Quick search from anywhere

**PRO Features**:
- Vector/semantic search using embeddings
- AI-powered search suggestions
- Entity-based search filters
- Search analytics and insights

### Task Management
**Description**: Integrated todo system for knowledge workers
**Technical Implementation**: Go backend with PostgreSQL, recurring task engine
**User Experience**:
- Create with 't' keyboard shortcut
- Today's task counter in sidebar
- Quick task creation dialog
- Priority and scheduling system

**Features**:
- Task creation and management
- Due dates and scheduling
- Priority levels (high, normal, low)
- Recurring task patterns
- Task tagging and categorization
- Today view with task counter
- Task completion tracking

### File Management
**Description**: Upload and organize supporting documents
**Technical Implementation**: S3-compatible storage with card attachment system
**User Experience**:
- Drag-and-drop file uploads
- File previews and thumbnails
- Attachment to cards
- File search capabilities

**Features**:
- File upload and storage
- Image, PDF, and document support
- File organization and tagging
- Attachment to cards
- File search and filtering
- Download and sharing

## PRO Features

### Entity Recognition
**Description**: Automatic extraction and management of people, places, concepts
**Technical Implementation**: LLM-powered NLP pipeline with entity database
**User Experience**:
- Automatic entity detection in content
- Entity cards and profiles
- Entity relationship mapping
- Manual entity creation and editing

**Features**:
- Named entity recognition (people, places, organizations)
- Entity relationship mapping
- Entity cards with metadata
- Entity-based navigation
- Entity frequency analysis
- Manual entity management

### Facts System
**Description**: Structured data extraction and management
**Technical Implementation**: JSON storage with schema validation
**User Experience**:
- Fact extraction from content
- Structured fact display
- Fact search and filtering
- Fact templates and schemas

**Features**:
- Structured fact extraction
- Fact categorization and tagging
- Fact relationships and linking
- Fact templates and schemas
- Fact verification and sources
- Fact export and import

### Memory System
**Description**: Personal knowledge retention and recall
**Technical Implementation**: Spaced repetition algorithms with usage analytics
**User Experience**:
- Knowledge retention tracking
- Personalized review schedules
- Memory performance analytics
- Adaptive learning algorithms

**Features**:
- Spaced repetition scheduling
- Knowledge retention tracking
- Memory performance analytics
- Adaptive review intervals
- Memory card creation
- Progress visualization

### Vector Search
**Description**: Semantic search using embeddings
**Technical Implementation**: pgvector with OpenAI embeddings, Typesense integration
**User Experience**:
- Semantic similarity search
- Related content discovery
- Concept-based navigation
- Search by meaning, not just keywords

**Features**:
- Semantic similarity search
- Content embeddings and clustering
- Related content recommendations
- Concept-based discovery
- Similar card suggestions
- Embedding visualization

### Content Analysis
**Description**: AI-powered insights and patterns
**Technical Implementation**: LLM processing pipeline with structured output
**User Experience**:
- Automatic content summaries
- Theme and pattern detection
- Content insights and recommendations
- Writing analysis and feedback

**Features**:
- Content summarization
- Theme extraction and analysis
- Writing style analysis
- Content recommendations
- Pattern recognition
- Insight generation

## Technical Features

### Self-Hosting
**Description**: Deploy on your own infrastructure
**Technical Implementation**: Docker containers with database migrations
**User Experience**:
- One-command deployment
- Environment configuration
- Data migration tools
- Backup and restore

**Features**:
- Docker-based deployment
- Database migrations
- Environment configuration
- SSL/TLS setup
- Backup and restore utilities
- Update management

### API Access
**Description**: Programmatic integration capabilities
**Technical Implementation**: RESTful API with JWT authentication
**User Experience**:
- API key management
- Documentation and examples
- SDK availability
- Webhook support

**Features**:
- RESTful API endpoints
- JWT authentication
- Rate limiting and quotas
- API documentation
- SDK libraries
- Webhook notifications

### Keyboard Shortcuts
**Description**: Efficient power-user workflows
**Technical Implementation**: JavaScript event handling with React
**User Experience**:
- Global keyboard shortcuts
- Context-sensitive shortcuts
- Customizable key bindings
- Shortcut help interface

**Features**:
- Global shortcuts (c, t, s)
- Context-sensitive shortcuts
- Shortcut customization
- Help documentation
- Accessibility support
- Vim-style navigation

### Templates
**Description**: Reusable card templates with variables
**Technical Implementation**: Template engine with variable substitution
**User Experience**:
- Template library
- Variable placeholders
- Quick template application
- Template sharing and import

**Features**:
- Template creation and management
- Variable substitution
- Template categories and tags
- Template sharing
- Quick template application
- Template versioning

## User Interface Features

### Responsive Design
**Description**: Optimized for all screen sizes
**Technical Implementation**: CSS Grid and Flexbox with React responsive utilities
**User Experience**:
- Mobile-first design
- Touch-friendly interactions
- Responsive layouts
- Adaptive navigation

### Dark Mode
**Description**: Eye-friendly dark theme
**Technical Implementation**: CSS custom properties with theme switching
**User Experience**:
- Automatic theme detection
- Manual theme toggle
- Consistent dark styling
- Accessibility compliance

### Real-time Updates
**Description**: Live synchronization across sessions
**Technical Implementation**: WebSocket connections with optimistic updates
**User Experience**:
- Instant updates
- Conflict resolution
- Offline support
- Sync indicators

### Progressive Web App
**Description**: App-like experience in the browser
**Technical Implementation**: Service worker with caching strategy
**User Experience**:
- Offline functionality
- Install prompts
- App-like navigation
- Push notifications

## Data and Privacy Features

### Data Ownership
**Description**: Complete control over your information
**Technical Implementation**: Self-hosted deployment options with data export
**User Experience**:
- Data export capabilities
- Migration tools
- Ownership transparency
- Privacy controls

### Encryption
**Description**: Data protection at rest and in transit
**Technical Implementation**: TLS for transport, AES for storage
**User Experience**:
- Transparent encryption
- Security indicators
- Privacy dashboard
- Audit logs

### Backup and Sync
**Description**: Reliable data protection and synchronization
**Technical Implementation**: Automated backups with versioning
**User Experience**:
- Automatic backups
- Manual backup triggers
- Restore capabilities
- Sync status indicators

## Integration Features

### Import/Export
**Description**: Data portability and migration
**Technical Implementation**: JSON, Markdown, and CSV format support
**User Experience**:
- Import from other tools
- Export to standard formats
- Bulk operations
- Migration wizards

### Third-party Integrations
**Description**: Connect with external tools and services
**Technical Implementation**: REST API with OAuth authentication
**User Experience**:
- Service connections
- Data synchronization
- Workflow automation
- Integration marketplace

### Webhook Support
**Description**: Real-time notifications and automation
**Technical Implementation**: HTTP callbacks with payload customization
**User Experience**:
- Event configuration
- Payload customization
- Delivery monitoring
- Retry mechanisms