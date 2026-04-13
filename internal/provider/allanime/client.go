package allanime

import (
	"context"
	"net/http"
	"time"

	"github.com/christophercuongkim/ani-cli-go/internal/config"
	"github.com/christophercuongkim/ani-cli-go/internal/provider"
	"github.com/hasura/go-graphql-client"
	"github.com/rs/zerolog/log"
)

const apiURL = "https://api.allanime.day/api"

// Compile-time interface compliance check
var _ provider.AnimeProvider = (*AllAnimeService)(nil)

type AllAnimeService struct {
	client *graphql.Client
	cfg    *config.Config
}

type headerTransport struct {
	base http.RoundTripper
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Referer", "https://allmanga.to")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0")
	return t.base.RoundTrip(req)
}

func NewAllAnimeService(cfg *config.Config) *AllAnimeService {
	httpClient := &http.Client{
		Transport: &headerTransport{base: http.DefaultTransport},
		Timeout:   30 * time.Second,
	}
	return &AllAnimeService{
		client: graphql.NewClient(apiURL, httpClient),
		cfg:    cfg,
	}
}

func (s *AllAnimeService) SearchAnime(ctx context.Context, query string, opts provider.SearchOptions) (*provider.SearchResult, error) {
	var queryStruct SearchQuery
	searchInput := SearchInput{
		Query:        graphql.String(query),
		AllowAdult:   graphql.Boolean(s.cfg.Search.AllowAdult),
		AllowUnknown: graphql.Boolean(s.cfg.Search.AllowUnknown),
	}
	vars := SearchVariables{
		Search:          searchInput,
		Limit:           graphql.Int(s.cfg.Search.Limit),
		Page:            graphql.Int(opts.Page),
		TranslationType: opts.TranslationType,
		CountryOrigin:   CountryOrigin(s.cfg.Search.CountryOrigin),
	}

	err := s.client.Query(ctx, &queryStruct, vars.ToMap())
	if err != nil {
		log.Debug().AnErr("error", err).Str("query", query).Msg("query failed.")
		return nil, err
	}

	result := provider.SearchResult{}
	result.HasMore = len(queryStruct.Shows.Edges) == s.cfg.Search.Limit
	result.Page = opts.Page

	providerShows := make([]provider.Show, len(queryStruct.Shows.Edges))
	for i, show := range queryStruct.Shows.Edges {
		providerShows[i] = provider.Show{
			ID:   string(show.ID),
			Name: string(show.Name),
			AvailableEpisodes: provider.AvailableEpisodes{
				Sub: int(show.AvailableEpisodes.Sub),
				Dub: int(show.AvailableEpisodes.Dub),
				Raw: int(show.AvailableEpisodes.Raw),
			},
		}
	}

	result.Shows = providerShows

	return &result, nil
}

func (s *AllAnimeService) GetEpisodeList(ctx context.Context, showID string) (*provider.EpisodeList, error) {
	// TODO: implement
	return nil, ErrNotImplemented
}

func (s *AllAnimeService) GetEpisodeSources(ctx context.Context, showID string, translationType provider.TranslationType, episode string) (*provider.Episode, error) {
	// TODO: implement
	return nil, ErrNotImplemented
}
