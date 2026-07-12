# SuperRead

Subscribe to RSS feeds. Auto-summarize everything into a daily briefing.

## Features

- Add RSS feeds with OPML batch import support
- Scheduled fetching to detect new articles
- One-sentence summarization for each article
- Daily briefing aggregating updates from all sources
- Smart deduplication when multiple sources cover the same event
- Read/unread tracking, favorites, tag categories

## Architecture

```text
Browser → React Frontend (Oxelia51 unified UI)
  → Go Backend (RSS Fetcher + Summarizer + Dedup)
  → PostgreSQL
```

## Installation

Integrated into the Oxelia51 platform. See [Oxelia51 deployment guide](https://github.com/XiaoleC05/Oxelia51).

## Usage

1. Visit [oxelia51.com](https://oxelia51.com), register and sign in
2. Open SuperRead from the tools menu
3. Add RSS sources and API key in settings
4. Check back daily for your briefing

## Contributing

1. Fork → 2. Feature branch → 3. Commit → 4. Push → 5. PR

## License

MIT License
