package provider

import "context"

// AnimeProvider interface for anime data sources
type AnimeProvider interface {
	// SearchAnime searches for anime by query string
	SearchAnime(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error)

	// GetEpisodeList returns available episodes for a show
	GetEpisodeList(ctx context.Context, showID string) (*EpisodeList, error)

	// GetEpisodeSources returns streaming sources for an episode
	GetEpisodeSources(ctx context.Context, showID string, translationType TranslationType, episode string) (*Episode, error)
}
