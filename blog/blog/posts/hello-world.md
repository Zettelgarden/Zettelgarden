---
title: "Creating This Blog: An Iterative Process"
date: "2024-11-29"
tags: ["react", "typescript", "blog", "tailwind"]
author: Nick Savage
excerpt: "How I built this blog through iterative development with Claude"
---

Today I'll share how I created this blog using React and TypeScript. It's a simple
setup that allows writing posts in markdown and displaying them on the web.

### Why a new blog?

Some readers may know that I am already blogging on [Substack](https://nsavage.substack.com). I decided that I wanted to build my own blogging platform for Zettelgarden mostly because I have a somewhat diverse audience of readers on Substack, including my mother. I was worried that if I posted in as much detail on Substack as I really wanted to about Zettelgarden development, I'd bore my audience to tears.

Having a dedicated development blog allows me to:
- Write technical content without worrying about the audience fit
- Document the development process in detail
- Keep the technical discussions separate from my general writing
- Practice what I preach by building in public

### Key Features

One of the main decisions was to keep things simple. This blog has:
- No database required - all posts are markdown files in the repository
- Simple, clean typography using Tailwind's typography plugin
- Fast loading times due to minimal architecture
- Easy writing experience using markdown
- Tag support for categorization

### Development Process with Claude

Building this blog was an interesting exercise in AI pair programming. Working with Claude (an AI assistant) helped in several ways:

1. **Iterative Development**: We broke down the project into small, manageable steps
2. **Quick Problem Solving**: When issues came up (like routing problems or styling conflicts), Claude could quickly suggest solutions
3. **Code Review**: Claude could spot potential issues and suggest improvements
4. **Architecture Decisions**: We discussed different approaches and their trade-offs

The development process typically went like this:
1. I'd describe what I wanted to achieve
2. Claude would suggest an implementation
3. I'd try it and report back any issues
4. We'd iterate until the feature worked as intended

Claude actually came me good suggestions, and generally didn't lead me astray. My conversation with it must be over the context window as well, and it is still giving good answers.

### Core Architecture

The core of the blog is surprisingly simple. Here's how it works:

1. **Reading Posts**: We use Vite's `import.meta.glob` to read markdown files:
```typescript
const postFiles = import.meta.glob('./posts/*.md', { as: 'raw' });
```
2. Parsing Metadata : We use gray-matter to parse frontmatter:
     
```typescript
const { data, content } = matter(fileContent);
 ```
 
3. Rendering : We use react-markdown to render the content:
     

```typescript
<ReactMarkdown>{post.content}</ReactMarkdown>
 ```
 
Everything else is just gravy around that.
 
### Current Challenges 
#### Deployment for New Posts 

One current limitation is that the site needs to be redeployed whenever I add a new blog post. This is because the markdown files are bundled during the build process. Some potential solutions I'm considering: 

- Moving posts somewhere else other than in repo
- Moving posts inside the database, so I don't need to manually repo changes
     

### Benefits of Markdown 

Using markdown for posts has been a great decision because: 

- Version Control : Posts can be tracked in git along with the code
- Simple Editing : Can use any text editor or IDE
- Preview Support : Most editors have markdown preview
- Portable : Posts can easily be moved to another platform if needed
- Code Blocks : Great support for sharing code snippets with syntax highlighting
     

### Looking Forward 

Future improvements I'm considering: 

- Automated deployments for new posts
- Better code syntax highlighting
- Dark mode support
- RSS feed support
- This blog lacks decent logging support
  - I want to be able to know things like which posts are being read, where are readers coming from, etc.
     
### Conclusion 

Building a custom blog has been a valuable learning experience. Working with Claude demonstrated how AI can be a useful tool in the development process, helping to solve problems and make decisions while keeping the implementation simple and maintainable. 

The end result is exactly what I needed: a simple, fast blog where I can write about technical topics without worrying about boring non-technical readers. The markdown-based approach makes writing new posts as simple as creating a new file and pushing to git. 

If you're interested in the technical details or want to see how something specific works, feel free to check out the [repository](https://github.com/NickSavage/Zettelgarden) or reach out to me directly at [nick@zettelgarden.com](nick@zettelgarden.com).
