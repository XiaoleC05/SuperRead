# SuperRead

Subscribe to RSS feeds. Auto-summarize everything into a daily briefing.

## Features

- Add RSS feeds with OPML batch import support
- Scheduled fetching every 30 minutes to detect new articles
- One-sentence summarization for each article
- Daily briefing aggregating updates from all sources
- Smart deduplication when multiple sources cover the same event
- Read/unread tracking, favorites, tag categories, read-later
- Platform notifications and optional email digests

## Architecture

```text
Browser
  ↓
React Frontend (Oxelia51 unified UI)
  ↓
Go Backend
  ├── RSS Fetcher (periodic cron jobs)
  ├── Summarizer (user-provided API key)
  └── Dedup Engine
  ↓
PostgreSQL / SQLite (feeds, articles, user data)
```

The online version runs on the Oxelia51 platform. The scheduler periodically fetches RSS sources, the dedup engine merges duplicate content, and the summarizer uses the user's own API key. The desktop version uses SQLite for storage.

## Requirements

- Online: Oxelia51 platform (Go, PostgreSQL, React)
- Desktop: standalone executable, no runtime dependencies
- API key for external model access

## Installation

### Desktop

Download `SuperRead.exe` from [GitHub Releases](https://github.com/XiaoleC05/SuperRead/releases).

### Online

Integrated into the Oxelia51 platform. See [Oxelia51 deployment guide](https://github.com/XiaoleC05/Oxelia51).

## Usage

### Online

1. Visit [oxelia51.com](https://oxelia51.com), register and sign in
2. Open SuperRead from the tools menu
3. Add RSS sources and enter your API key in settings
4. Check back daily for your briefing

### Desktop

1. Double-click `SuperRead.exe` to start
2. Add RSS sources and API key. All data is stored locally.

## Roadmap

- [ ] RSS source management and fetching
- [ ] Summarization
- [ ] Daily briefing display
- [ ] Smart deduplication

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/xxx`)
3. Commit your changes (`git commit -m 'Add xxx'`)
4. Push the branch (`git push origin feature/xxx`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.
