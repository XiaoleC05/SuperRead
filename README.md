# SuperRead — AI-Powered News Briefing

> Subscribe to your favorite sources. Get an AI-summarized daily brief. Spend 5 minutes instead of 2 hours.

## Why SuperRead?

Keeping up with tech news, industry blogs, and developer updates means checking dozens of sites — or drowning in an RSS reader with hundreds of unread articles. You don't need every full article; you need to know what happened today.

**SuperRead** subscribes to the sources you care about, fetches new content on a schedule, and uses AI to summarize everything into a concise daily briefing. Read the summaries in minutes. Click through to the original only when something matters.

## Features

| Feature | What It Does |
|---------|-------------|
| **Source Management** | Add RSS feeds, auto-discover feeds from URLs, or import your OPML file |
| **Scheduled Fetching** | Background updates every 30 minutes — always fresh |
| **AI Summarization** | Each article distilled into a one-sentence summary |
| **Daily Briefing** | All of today's updates from all sources, in one scrollable feed |
| **Smart Deduplication** | Same story from multiple sources? Merged into one entry |
| **Reading Management** | Mark as read/unread, star favorites, tag categories, save for later |
| **Notifications** | Platform alerts + optional email digests when new content arrives |
| **Original Links** | Every summary includes a link to the full article |

## Multi-User

Each user has their own independent subscription list and briefing feed. No social features — this is a pure reading tool.

## Tech Stack

| Environment | Backend | Database | Frontend | Special |
|-------------|---------|----------|----------|---------|
| Online (Oxelia51) | Go + cron | PostgreSQL | React | RSS parser + LLM API |
| Desktop (exe) | Go + cron | SQLite | Embedded React | Same, packaged as exe |

- Scheduled fetching uses Go's built-in cron — no external scheduler
- AI summarization requires your own API key from any supported provider

## API Key

SuperRead uses your own LLM API key for summarization. The key is stored locally. You control the cost — adjust fetch frequency and summary length to fit your budget.

## Getting Started

### Online (via Oxelia51)

1. Visit [oxelia51.com](https://oxelia51.com) and sign in
2. Open SuperRead from the tools menu
3. Add your RSS sources and enter your API key
4. Check back for your daily briefing

### Desktop (exe)

1. Download `SuperRead.exe` from [GitHub Releases](https://github.com/XiaoleC05/SuperRead/releases)
2. Run the executable — starts a local web interface
3. Add sources, enter API key, everything runs locally

## Status

Concept phase. Development not yet started.
