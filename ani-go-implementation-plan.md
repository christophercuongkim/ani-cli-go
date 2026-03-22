# ani-go: Implementation Plan for Porting ani-cli to Go

> A comprehensive implementation plan for building a Go port of [ani-cli](https://github.com/pystardust/ani-cli) with significant enhancements over the original bash script.

---

## Project Overview

**Goal:** Port ani-cli from a ~600-line bash script into a compiled, cross-platform Go binary that eliminates external dependencies (fzf, curl, sed, grep) and adds first-class features like an embedded TUI, built-in media player integration, watchlist sync, and a proper config system.

**Source Repo:** <https://github.com/pystardust/ani-cli>

**Original Dependencies Eliminated:** `fzf`, `curl`, `sed`, `grep`, `aria2`, `yt-dlp`, `patch`

---

## Phase Execution Order

The 12 phases are ordered by dependency graph — each phase builds on what came before it.

| # | Phase | Depends On | Effort |
|---|-------|-----------|--------|
| 1 | Project Scaffold & Config | — | S |
| 2 | AllAnime API Client | Phase 1 | M |
| 3 | Scraper & Link Extractor | Phase 2 | L |
| 4 | libmpv cgo Integration | Phase 1 | L |
| 5 | SQLite History | Phase 1 | M |
| 6 | Download Manager | Phase 3 | M |
| 7 | AniList / MAL Integration | Phase 5 | M |
| 8 | ani-skip & Schedule API | Phase 2, 4 | S |
| 9 | Syncplay | Phase 4 | M |
| 10 | Bubbletea TUI | Phases 2-9 | XL |
| 11 | Self-Update | Phase 1 | S |
| 12 | Packaging & NixOS Flake | All | M |

**Effort Key:** S = 1-2 days, M = 3-5 days, L = 1-2 weeks, XL = 2-3 weeks

---

## Phase 1: Project Scaffold & Config

### Goal
Set up the Go module, directory layout, CLI flag parsing, and TOML configuration system.

### Directory Structure
```
ani-go/
├── cmd/
│   └── ani-go/
│       └── main.go             # Entrypoint, flag parsing
├── internal/
│   ├── config/
│   │   └── config.go           # TOML config loader + defaults
│   ├── api/
│   │   └── allanime.go         # Phase 2
│   ├── scraper/
│   │   └── extractor.go        # Phase 3
│   ├── player/
│   │   ├── mpv.go              # Phase 4 (libmpv)
│   │   └── subprocess.go       # Phase 4 (fallback)
│   ├── history/
│   │   └── sqlite.go           # Phase 5
│   ├── download/
│   │   └── manager.go          # Phase 6
│   ├── trackers/
│   │   ├── anilist.go          # Phase 7
│   │   └── mal.go              # Phase 7
│   ├── skip/
│   │   └── aniskip.go          # Phase 8
│   ├── syncplay/
│   │   └── client.go           # Phase 9
│   └── tui/
│       ├── app.go              # Phase 10
│       ├── search.go
│       ├── episodes.go
│       └── player.go
├── providers/
│   └── providers.yaml          # Provider config for maintainability
├── config.example.toml
├── go.mod
├── go.sum
├── flake.nix                   # Phase 12
├── flake.lock
├── Makefile
└── README.md
```

### Config File (`~/.config/ani-go/config.toml`)
```toml
[general]
player = "libmpv"           # "libmpv" | "mpv" | "vlc" | "iina"
quality = "best"            # "best" | "worst" | "1080" | "720" | "480" | "360"
mode = "sub"                # "sub" | "dub"
skip_intro = false
detach_player = false

[downloads]
directory = "~/Videos/Anime"
concurrent_fragments = 16

[history]
db_path = "~/.local/share/ani-go/history.db"

[anilist]
enabled = false
token = ""

[mal]
enabled = false
client_id = ""

[syncplay]
server = ""
room = ""
username = ""

[tui]
theme = "default"           # "default" | "catppuccin" | "dracula"
```

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **BurntSushi/toml** | `github.com/BurntSushi/toml` | TOML config parsing |
| **cobra** | `github.com/spf13/cobra` | CLI framework with subcommands & flags |

### Key Implementation Notes
- CLI flags override TOML config values (cobra + viper pattern, or manual merge)
- XDG base directory compliance: config in `$XDG_CONFIG_HOME/ani-go/`, data in `$XDG_DATA_HOME/ani-go/`, state in `$XDG_STATE_HOME/ani-go/`
- Build tags: `libmpv` and `nolibmpv` for toggling cgo player backend

---

## Phase 2: AllAnime API Client

### Goal
Implement the GraphQL client that searches anime, lists episodes, and fetches stream source IDs from the AllAnime API.

### How ani-cli Does It
The original uses `curl` with URL-encoded GraphQL queries against `https://api.allanime.day/api` (or mirror), parsing JSON responses with `sed`.

### Key Endpoints
- **Search:** GraphQL query with `$search`, `$limit`, `$translationType` (sub/dub)
- **Episode List:** Query with `$showId` → returns `availableEpisodesDetail`
- **Episode Sources:** Query with `$showId`, `$episodeString`, `$translationType` → returns provider source IDs

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **net/http** | `net/http` (stdlib) | HTTP requests |
| **encoding/json** | `encoding/json` (stdlib) | JSON marshal/unmarshal |
| **graphql-go/graphql** | — | *Not needed* — raw POST with JSON body is simpler for this API |

### Key Implementation Notes
- Use a custom `User-Agent` string matching ani-cli's agent to avoid blocks
- Set the `Referer` header to `$allanime_refr` (currently `https://allanime.to`)
- Define Go structs for API responses (search results, episode lists, source lists)
- Handle API mirror rotation if the primary endpoint goes down
- Implement retry logic with exponential backoff

### Improvement: Concurrent Provider Resolution
When ani-cli fetches episode sources, it gets multiple provider IDs and resolves them sequentially. In Go, resolve all providers concurrently with goroutines + `errgroup`:

```go
import "golang.org/x/sync/errgroup"
```

| Library | Import Path | Purpose |
|---------|------------|---------|
| **errgroup** | `golang.org/x/sync/errgroup` | Concurrent provider resolution |

---

## Phase 3: Scraper & Link Extractor

### Goal
Decrypt and extract actual video stream URLs from the provider source data returned by the AllAnime API.

### How ani-cli Does It
1. Fetches the provider embed page
2. Decrypts obfuscated URLs (hex decoding, character shifting)
3. Extracts M3U8/MP4 links from the decrypted data
4. Parses quality tiers from the response

### Provider Config (`providers.yaml`)
Instead of hardcoding scraping logic, use a YAML config for maintainability:
```yaml
providers:
  - name: "Luf-mp4"
    type: "direct"
    priority: 1
    decrypt: "hex_shift"
  - name: "S-mp4"
    type: "direct"
    priority: 2
    decrypt: "hex_shift"
  - name: "Kir"
    type: "hls"
    priority: 3
    decrypt: "none"
```

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **goquery** | `github.com/PuerkitoBio/goquery` | HTML parsing for embed pages |
| **gopkg.in/yaml.v3** | `gopkg.in/yaml.v3` | Provider config parsing |
| **regexp** | `regexp` (stdlib) | Pattern extraction from scripts |

### Key Implementation Notes
- Implement the hex decode + character shift decryption (ani-cli's `sed` pipeline that converts hex pairs and shifts by a key)
- Support both direct MP4 links and M3U8/HLS manifests
- Parse quality options from `resolutionStr` fields
- The scraping logic is the most fragile part — the provider YAML approach makes it easy to update when AllAnime changes their obfuscation

---

## Phase 4: libmpv cgo Integration

### Goal
Embed mpv as a library via cgo for native playback control, with a subprocess fallback for systems without libmpv headers.

### Primary: libmpv via cgo

| Library | Import Path | Purpose |
|---------|------------|---------|
| **go-mpv** | `github.com/gen2brain/go-mpv` | Go bindings for libmpv (supports cgo and purego/nocgo) |

### Features Enabled by libmpv
- Programmatic playback control (play, pause, seek, volume)
- Property observation (current position, duration, chapter)
- Event handling (file loaded, end of file, seek)
- IPC for next-episode prefetch triggers
- Resume position tracking at timestamp level
- Skip intro/outro integration without external scripts

### Fallback: Subprocess mpv/vlc
Use build tags to compile without cgo:
```go
//go:build nolibmpv

package player

// Falls back to os/exec subprocess control
```

| Library | Import Path | Purpose |
|---------|------------|---------|
| **os/exec** | `os/exec` (stdlib) | Subprocess player launch |
| **blang/mpv** | `github.com/blang/mpv` | mpv IPC via JSON socket (for subprocess mode) |

### Key Implementation Notes
- **NixOS Validation:** Test cgo compilation early — NixOS needs `libmpv-dev` headers available at build time. Add to the flake's `buildInputs`
- The `gen2brain/go-mpv` library supports a `nocgo` build tag that uses purego, which is useful for cross-compilation
- Define a `Player` interface that both backends implement:
  ```go
  type Player interface {
      Play(url string, opts PlayOpts) error
      Pause() error
      Resume() error
      Seek(seconds float64) error
      GetPosition() (float64, error)
      OnEvent(EventType, func(Event)) error
      Close() error
  }
  ```
- Set `--force-media-title` for proper episode naming in the player UI
- Set `--referrer` header for streams that require it

---

## Phase 5: SQLite History

### Goal
Replace ani-cli's flat-file history with a proper SQLite database that tracks watch position at the timestamp level, not just episode number.

### Schema
```sql
CREATE TABLE watch_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    allanime_id TEXT NOT NULL,
    title TEXT NOT NULL,
    episode TEXT NOT NULL,
    position_seconds REAL DEFAULT 0,
    duration_seconds REAL DEFAULT 0,
    completed BOOLEAN DEFAULT FALSE,
    last_watched DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(allanime_id, episode)
);

CREATE TABLE anime_metadata (
    allanime_id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    mal_id INTEGER,
    anilist_id INTEGER,
    total_episodes INTEGER,
    cover_url TEXT,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_history_last_watched ON watch_history(last_watched DESC);
CREATE INDEX idx_history_allanime ON watch_history(allanime_id);
```

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **go-sqlite3** | `github.com/mattn/go-sqlite3` | SQLite driver (cgo) |
| **modernc sqlite** | `modernc.org/sqlite` | Pure Go SQLite (no cgo alternative) |

### Key Implementation Notes
- Default to `modernc.org/sqlite` for zero cgo dependency on the history side, unless the libmpv build tag already requires cgo anyway
- Store resume position: when the player fires a position event, periodically write it back (every 30 seconds or on pause/quit)
- Mark episode as `completed` when position > 90% of duration
- Provide `--continue` / `-c` flag that picks up the last unwatched or in-progress episode
- Migration support: import from ani-cli's existing `~/.local/state/ani-cli/` history file

---

## Phase 6: Download Manager

### Goal
Replace the `aria2` / `yt-dlp` dependency with a native Go download manager supporting concurrent fragment downloads for HLS streams and direct file downloads.

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **grab** | `github.com/cavaliergopher/grab/v3` | HTTP file downloads with progress |
| **m3u8** | `github.com/grafov/m3u8` | M3U8/HLS playlist parsing |
| **mpb** | `github.com/vbauerster/mpb/v8` | Terminal progress bars |

### Key Implementation Notes
- For direct MP4 downloads: single-file download with progress bar
- For HLS streams: parse M3U8 manifest, download segments concurrently (configurable worker count, default 16), then concatenate with FFmpeg or pure Go muxing
- Support resume/retry on network failure
- File naming: `{title}/Episode_{number}.mp4`
- Quality selection before download (reuse Phase 3's quality parsing)
- **FFmpeg dependency note:** HLS segment concatenation may still need FFmpeg for remuxing. Consider bundling a static FFmpeg binary or documenting it as an optional dependency for downloads only.

---

## Phase 7: AniList / MAL Integration

### Goal
Sync watch progress with AniList and MyAnimeList via OAuth2, enabling two-way watchlist management.

### AniList

| Library | Import Path | Purpose |
|---------|------------|---------|
| **oauth2** | `golang.org/x/oauth2` | OAuth2 authorization flow |
| **net/http** | `net/http` (stdlib) | AniList GraphQL API |

- API: `https://graphql.anilist.co`
- Auth: OAuth2 Authorization Code flow → opens browser → local callback server on `localhost:PORT`
- Store token in config or OS keyring

### MyAnimeList

| Library | Import Path | Purpose |
|---------|------------|---------|
| **oauth2** | `golang.org/x/oauth2` | OAuth2 with PKCE |

- API: `https://api.mymal.net/v2/` (or official MAL API)
- Auth: OAuth2 with PKCE (no client secret needed)
- MAL requires a registered Client ID

### Sync Logic
1. When an episode is marked `completed` in SQLite history (Phase 5), auto-increment the tracker's episode count
2. On startup, optionally pull the tracker's watchlist to populate "continue watching" suggestions
3. Map AllAnime IDs to MAL/AniList IDs using the `anime_metadata` table

### Key Implementation Notes
- The ID mapping between AllAnime ↔ MAL ↔ AniList is the trickiest part. AllAnime's API may return `malId` directly; otherwise use a mapping service or Jikan API
- Token refresh handling: store refresh tokens securely
- Rate limiting: AniList allows 90 requests/minute; MAL is stricter

---

## Phase 8: ani-skip & Schedule API

### Goal
Integrate intro/outro skip timestamps and next-episode countdown.

### ani-skip Integration
- API: `https://api.aniskip.com/v2/skip-times/{mal_id}/{episode}?types[]=op&types[]=ed`
- Returns start/end timestamps for opening and ending sequences
- Feed timestamps to mpv via `--script-opts=skip-op_start=X,skip-op_end=Y` or programmatically via libmpv chapter/seek control

### Schedule API
- API: `https://animeschedule.net/api/v3/anime?q={query}`
- Show countdown to next episode airing
- Display in TUI as "Next episode in X days, Y hours"

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **net/http** | `net/http` (stdlib) | API requests |
| **encoding/json** | `encoding/json` (stdlib) | Response parsing |

### Key Implementation Notes
- Requires MAL ID mapping (from Phase 7's metadata table or AllAnime's API response)
- With libmpv: observe the `time-pos` property and auto-seek past the OP/ED ranges
- With subprocess mpv: pass `--script-opts` flags at launch
- The `--skip-title` override flag should still be supported for manual overrides

---

## Phase 9: Syncplay

### Goal
Enable watching anime in sync with friends via the Syncplay protocol.

### How It Works
- Syncplay uses a client-server model over TCP
- The client connects to a Syncplay server, joins a room, and relays play/pause/seek events
- Original ani-cli passes `--syncplay` to mpv which handles it

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **net** | `net` (stdlib) | TCP client for Syncplay protocol |
| **encoding/json** | `encoding/json` (stdlib) | Syncplay message format |

### Key Implementation Notes
- With libmpv: implement the Syncplay protocol directly in Go, controlling mpv playback programmatically in response to sync messages
- With subprocess mpv: rely on mpv's built-in `--syncplay` option (simpler but less control)
- Protocol: JSON messages over TCP — relatively simple to implement
- Config: server, room, username from `config.toml`

---

## Phase 10: Bubbletea TUI

### Goal
Replace the fzf dependency with a native terminal UI built on Bubbletea, providing a richer interactive experience.

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **bubbletea** | `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| **bubbles** | `github.com/charmbracelet/bubbles` | Pre-built components (text input, list, spinner, viewport, table) |
| **lipgloss** | `github.com/charmbracelet/lipgloss` | Styling and layout |
| **glamour** | `github.com/charmbracelet/glamour` | Markdown rendering (for descriptions) |

### TUI Screens

1. **Search Screen**
   - Text input with real-time search (debounced)
   - Filterable results list with anime title, episode count, type (sub/dub)
   - Fuzzy matching via bubbles' list filtering

2. **Episode Selection Screen**
   - Episode list with watched/unwatched indicators (from SQLite history)
   - Multi-select support for batch downloads
   - Resume position shown as progress bar per episode
   - "Continue from last" quick action

3. **Player Control Screen**
   - Now playing: title, episode, quality
   - Playback progress bar
   - Controls: next, previous, replay, change quality
   - ani-skip indicators (OP/ED regions highlighted on progress bar)

4. **History / Continue Watching Screen**
   - Recently watched anime sorted by last watched
   - Per-anime episode progress
   - AniList/MAL sync status indicators

5. **Settings Screen**
   - Quality preference
   - Sub/dub toggle
   - Player backend selection
   - Tracker authentication status

### Key Implementation Notes
- Follow Elm architecture: each screen is a `tea.Model` with `Init()`, `Update()`, `View()`
- Use lipgloss for consistent theming (support catppuccin, dracula, etc.)
- Debounce search input (300ms) to avoid hammering the API
- The TUI should gracefully degrade: if terminal doesn't support colors, fall back to basic rendering
- Support `--rofi` and `--dmenu` flags for external menu compatibility (matching ani-cli's `$ANI_CLI_EXTERNAL_MENU`)

---

## Phase 11: Self-Update

### Goal
Provide a built-in update mechanism similar to ani-cli's `-U` flag.

### Libraries

| Library | Import Path | Purpose |
|---------|------------|---------|
| **selfupdate** | `github.com/minio/selfupdate` | In-place binary replacement |
| **go-github** | `github.com/google/go-github/v60` | GitHub Releases API |

### Key Implementation Notes
- Check GitHub Releases for the latest version tag
- Download the appropriate binary for `GOOS`/`GOARCH`
- Verify checksum before replacing
- Show changelog/release notes in the TUI
- Skip self-update on NixOS (detect via `/etc/NIXOS` or `NIX_PROFILES` env) — Nix manages updates via the flake

---

## Phase 12: Packaging & NixOS Flake

### Goal
Provide cross-platform distribution with a first-class NixOS flake as a packaging target.

### Build & Release

| Tool | Purpose |
|------|---------|
| **goreleaser** | Cross-platform binary builds, checksums, GitHub Release publishing |
| **Makefile** | Local development build targets |

### NixOS Flake (`flake.nix`)
```nix
{
  description = "ani-go - A Go port of ani-cli";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.buildGoModule {
          pname = "ani-go";
          version = "0.1.0";
          src = ./.;
          vendorHash = ""; # Fill after first build
          buildInputs = [ pkgs.mpv-unwrapped ];
          nativeBuildInputs = [ pkgs.pkg-config ];
          tags = [ "libmpv" ];
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            mpv-unwrapped
            pkg-config
            sqlite
          ];
        };
      });
}
```

### Additional Packaging
- **AUR:** PKGBUILD for Arch Linux
- **Homebrew:** Formula for macOS
- **Docker:** Minimal image for headless download-only mode
- **Goreleaser config** (`.goreleaser.yml`): builds for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)

### Key Implementation Notes
- The NixOS flake should be validated early (during Phase 4) to catch cgo/libmpv build issues
- Provide a `nolibmpv` build for systems without libmpv: `go build -tags nolibmpv`
- Consider a `completions` subcommand that generates shell completions (bash, zsh, fish) for easy integration with dotfiles

---

## Complete Dependency Map

### Core Dependencies

| Library | Import Path | Used In | cgo? |
|---------|------------|---------|------|
| BurntSushi/toml | `github.com/BurntSushi/toml` | Config | No |
| cobra | `github.com/spf13/cobra` | CLI | No |
| bubbletea | `github.com/charmbracelet/bubbletea` | TUI | No |
| bubbles | `github.com/charmbracelet/bubbles` | TUI components | No |
| lipgloss | `github.com/charmbracelet/lipgloss` | TUI styling | No |
| glamour | `github.com/charmbracelet/glamour` | Markdown render | No |
| go-mpv | `github.com/gen2brain/go-mpv` | Player (libmpv) | Yes* |
| blang/mpv | `github.com/blang/mpv` | Player (IPC fallback) | No |
| go-sqlite3 | `github.com/mattn/go-sqlite3` | History (cgo) | Yes |
| modernc sqlite | `modernc.org/sqlite` | History (pure Go) | No |
| goquery | `github.com/PuerkitoBio/goquery` | HTML scraping | No |
| yaml.v3 | `gopkg.in/yaml.v3` | Provider config | No |
| grab | `github.com/cavaliergopher/grab/v3` | Downloads | No |
| m3u8 | `github.com/grafov/m3u8` | HLS parsing | No |
| mpb | `github.com/vbauerster/mpb/v8` | Progress bars | No |
| oauth2 | `golang.org/x/oauth2` | AniList/MAL auth | No |
| go-github | `github.com/google/go-github/v60` | Self-update | No |
| selfupdate | `github.com/minio/selfupdate` | Binary replacement | No |
| errgroup | `golang.org/x/sync/errgroup` | Concurrency | No |

\* `gen2brain/go-mpv` supports a `nocgo` build tag using purego

### Build Tools

| Tool | Purpose |
|------|---------|
| goreleaser | Release automation |
| golangci-lint | Linting |
| gotest | Testing |

---

## Improvements Over Original ani-cli

| Feature | ani-cli (bash) | ani-go |
|---------|---------------|--------|
| Dependencies | fzf, curl, sed, grep, aria2, yt-dlp | Single binary |
| UI | fzf / rofi / dmenu | Native Bubbletea TUI |
| Config | Environment variables | TOML config file |
| History | Flat file (episode only) | SQLite (with resume timestamps) |
| Player | Subprocess mpv/vlc | Embedded libmpv + subprocess fallback |
| Downloads | aria2 / yt-dlp | Built-in concurrent downloader |
| Provider resolution | Sequential | Concurrent (goroutines) |
| Tracker sync | None | AniList + MAL OAuth |
| Skip intro | External ani-skip script | Built-in aniskip API |
| Prefetch | None | Next-episode prefetch via mpv IPC |
| Packaging | Manual install / distro packages | goreleaser + NixOS flake |
| Cross-platform | Requires bash + GNU tools | Native binaries for all platforms |

---

## Development Milestones

### MVP (Phases 1-5) — Core Functionality
Search, browse, play, and track watch history. This is the minimum viable replacement for ani-cli.

### Feature Parity (Phases 6-9) — Full ani-cli Coverage
Downloads, tracker sync, intro skip, syncplay. At this point, ani-go matches or exceeds everything ani-cli can do.

### Polish (Phases 10-12) — Production Ready
Full TUI, self-update, cross-platform packaging. Ready for public release.

---

## NixOS-Specific Notes

Since you're developing on NixOS, keep these in mind:

1. **Phase 4 is the cgo validation checkpoint.** If libmpv + cgo builds cleanly in your Nix devshell, the rest of the project will be smooth.
2. **Add `mpv-unwrapped` and `pkg-config` to your devshell** — `mpv` (the wrapped version) won't expose headers for cgo.
3. **SQLite cgo:** If using `mattn/go-sqlite3`, you'll need `sqlite.dev` in `buildInputs`. Consider `modernc.org/sqlite` to avoid this.
4. **Disable self-update on NixOS:** Detect `/etc/NIXOS` and skip in-place binary updates. The flake handles it.
5. **Shell completions:** Generate zsh completions and point Home Manager to them from your dotfiles repo for cross-platform compatibility.
