---
title: "From Vector Search to Entity Processing: Evolving Zettelgarden's Connection Engine"
date: "2024-12-02"
tags: [ "vector-search", "entity-processing", "llms", "knowledge-management", "rag"]
author: Nick Savage
excerpt: "A deep dive into how Zettelgarden evolved from basic vector search to a sophisticated entity processing system, exploring the challenges and opportunities in augmenting human knowledge connections."
---

In this post, I want to talk about some of the recent changes to the processes of understanding semantic meaning behind Zettelgarden. I am going to discuss where I want to be going, the first attempts and where we are now.

I have been writing about it for the last few weeks, but Zettelgarden is a digital ‘zettelkasten’ that I have been working on as a personal project and for my own benefit. I have recently been experimenting with integrating LLMs in it to help build connections and for search. Check it out on Github: [https://github.com/NickSavage/Zettelgarden](https://github.com/NickSavage/Zettelgarden)

### Where do we want to go

Let's take a step back and remember the core purpose of a Zettelkasten: building meaningful connections between ideas. When I started my Zettelkasten journey on paper, I accumulated around 5,000 physical index cards before realizing I needed a different approach. This collection became one of the driving forces behind developing Zettelgarden - I needed a tool that could help manage and discover connections across this substantial body of knowledge. 

While my early experiments with vector search showed promising results [link to previous post], the fundamental challenge that inspired Zettelgarden remains: how to effectively manage connections across a large collection of notes. With paper cards, I noticed a pattern that many Zettelkasten users might find familiar. Making connections between recently added cards comes naturally - our minds easily spot relationships, contradictions, and supporting evidence between ideas we've encountered in the past few weeks or months. It's like working with a moving window of connections, where our recent thoughts are crystal clear, but older notes gradually fade into the background. 

This limitation isn't just a personal quirk - it's a real barrier to the Zettelkasten's potential, especially with paper systems. While I could easily weave together insights from my last few hundred cards, valuable connections with older notes often remained undiscovered. The real power of a Zettelkasten lies in its ability to surface unexpected connections across your entire body of knowledge, no matter when you created those notes. 

This is where we want to go: breaking free from being limited to only recent connections and unlocking the full potential of cross-pollination between ideas. Whether it's identifying thematic links, spotting contradictions, or finding supporting evidence, we need tools that can help us see across our entire knowledge base with the same clarity we have for our recent notes - a goal that has driven Zettelgarden's development from the start. 

### First attempts at vector search

My initial experiments with vector search felt almost magical. The system could easily find connections between cards that I hadn't noticed before, and the results were promising enough to show that this approach had real potential for expanding the scope of connections beyond just recent notes. 

However, these early attempts had some significant limitations. I started by calculating vectors for entire cards, which worked well enough for shorter notes but fell apart completely when dealing with longer content like podcast transcripts. These longer cards would primarily match with other long-form content rather than finding meaningful thematic connections. I suspect this was partly due to the context window limitations of the LLMs - when you feed in a long transcript, the model tends to focus mainly on the beginning, missing important content further down. 

This led me to experiment with chunking - breaking down cards into smaller pieces for vector calculation. After some trial and error, I settled on chunks of around 300 characters, usually working out to about a sentence or two. Each chunk gets its own vector representation, allowing for much more granular matching. 

The next evolution was adding reranking to the process. While vector search could find potentially relevant matches, I needed a way to sort through these results more intelligently. By using an LLM to evaluate the actual relevance of each match to the original query, we could surface the most meaningful connections rather than just the most mathematically similar ones. 

These improvements made a significant difference, but they also revealed new challenges around storage requirements, processing time, and the complexity of handling references between cards - challenges that would shape the next phase of development. 

### Entity processing
My latest approach focuses on identifying and managing "entities" - people, places, events, and other distinct concepts within cards. This was partly inspired by LightRAG's approach, though I've taken a different direction that I believe is more efficient. I've gone down this road because I still am not very happy with the way search is working. It has the semblance of quality, but I haven't found it very useful in practice while trying to make connections, since I am dogfooding this.

The basic concept is straightforward: use LLMs to extract meaningful entities from card content, then store and embed these entities separately. This creates a junction table that effectively maps relationships between cards based on shared entities. For example, if two cards mention "Kaiser Wilhelm II" (even if one refers to him as "Kaiser Wilhelm" and another as "Wilhelm II"), the system can recognize this as the same entity and establish a connection between those cards. 

The process works in three main steps: 

1. Entity Detection: An LLM examines the text and identifies potential entities
2. Duplicate Detection: The system checks if these entities already exist in the database, using vector search as a first pass to find potential matches, then using an LLM to confirm if they're truly the same entity
3. Storage and Linking: Normalized entities are stored in the database with links to their original cards

Unfortunately, the code behind this is largely prompts, not special magic. Our entity finding code looks like this:


```
You are an AI specialized in analyzing zettelkasten cards and extracting entities.
Follow these rules strictly:

1. Entity Types must be one of: person, concept, theory, book, software, place, organization, event, method

2. Entity Names should be:
   - Concise (1-5 words)
   - Properly capitalized
   - Specific enough to be unique
   - Consistent with academic/professional terminology

3. Descriptions should be:
   - 10-20 words maximum
   - Focus on relevance to the card's context
   - Include key relationships or significance
   - Objective and factual

4. Extract entities that are:
   - Explicitly mentioned in the text
   - Significant to the card's main ideas
   - Could be useful for connecting to other cards
   - Worth tracking as separate concepts

Please analyze this zettelkasten card and extract all meaningful entities:
    Title: %s
    Body: %s
    
    Return only a JSON array of entities matching this structure:
[
    
        {
            "name": "entity name",
            "description": "brief description",
            "type": "entity type"
        }
    ]
```


Then finding duplicates:

```
Entity 1:
Name: %s
Type: %s
Description: %s

Entity 2:
Name: %s
Type: %s
Description: %s

Return JSON in this format:
{
    "areSame": boolean,
    "explanation": "brief explanation of decision",
    "preferredEntity": "1" or "2"
}`,
```
### Analysis and What's Next

This approach is proving more storage-efficient than previous attempts. With about 9,000 cards, the database is currently around 260MB (about 30KB per card on average), though there's a power law distribution where some cards with many chunks and vectors take up significantly more space. For my own uses, this is fine, but if Zettelgarden takes off we're talking terabytes of floating point numbers being stored.

The entity processing system is still in its early stages, and there are several directions I'm excited to explore. First, I want to experiment with different ways of surfacing these entity connections to users - perhaps as subtle suggestions while writing, or as an optional layer of discovery alongside traditional search. I'm also interested in testing different LLM models for entity detection, as there might be a sweet spot between accuracy and processing costs that I haven't found yet. 

A particularly intriguing possibility is using the entity system to help identify gaps in knowledge. If multiple cards reference a specific concept or person, but there's no dedicated card exploring that entity in depth, that might be a valuable signal for where to focus future research and note-taking. The challenge will be implementing these features while maintaining Zettelgarden's core principle: augmenting rather than replacing human thought processes. 

You can check out Zettelgarden either on its [website](https://zettelgarden.com) or check it out on Github: [https://github.com/NickSavage/Zettelgarden](https://github.com/NickSavage/Zettelgarden)