# CLI Help Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add CLI Tool documentation section to the HelpPage
**Architecture:** Add new help content to existing helpContent.ts data file
**Tech Stack:** React, TypeScript, existing HelpSection components

---

## Task Structure

### Task 1: Add CLI Tool Help Content
**Files:**
- Modify: `zettelkasten-front/src/data/helpContent.ts`

**Step 1: Add CLI Tool section to helpSections array**
Add a new help section with:
- id: 'cli-tool'
- title: 'CLI Tool'
- icon: '⌨️'
- category: 'advanced'
- level: 'intermediate'
- order: 6 (after existing sections)
- content array with:
  - Overview text explaining zg CLI
  - Configuration instructions
  - Command reference table (using 'list' content type)
  - Tips section (using 'callout' content type)

**Step 2: Test the HelpPage renders new content**
- Start dev server
- Navigate to Help page
- Verify CLI Tool section appears in Advanced category
**Step 3: Commit changes**
```bash
git add zettelkasten-front/src/data/helpContent.ts
git commit -m "docs: add CLI tool help section to HelpPage"
```
