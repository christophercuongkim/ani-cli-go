package allanime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/christophercuongkim/ani-cli-go/internal/config"
	"github.com/christophercuongkim/ani-cli-go/internal/provider"
	"github.com/hasura/go-graphql-client"
)

// testConfig returns a config for testing
func testConfig() *config.Config {
	return &config.Config{
		Search: config.Search{
			AllowAdult:    false,
			AllowUnknown:  false,
			Limit:         40,
			CountryOrigin: "ALL",
		},
	}
}

// mockServer creates a test server that returns the given response
func mockServer(t *testing.T, response any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
}

// newTestService creates an AllAnimeService pointing to test server
func newTestService(serverURL string, cfg *config.Config) *AllAnimeService {
	return &AllAnimeService{
		client: graphql.NewClient(serverURL, nil),
		cfg:    cfg,
	}
}

func TestSearchAnime(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		opts           provider.SearchOptions
		serverResponse map[string]any
		wantShows      int
		wantHasMore    bool
		wantErr        bool
	}{
		{
			name:  "successful search with results",
			query: "naruto",
			opts: provider.SearchOptions{
				TranslationType: provider.TranslationSub,
				Page:            1,
			},
			serverResponse: map[string]any{
				"data": map[string]any{
					"shows": map[string]any{
						"edges": []map[string]any{
							{
								"_id":        "show-123",
								"name":       "Naruto",
								"__typename": "Show",
								"availableEpisodes": map[string]any{
									"sub": 220,
									"dub": 220,
									"raw": 220,
								},
							},
							{
								"_id":        "show-456",
								"name":       "Naruto Shippuden",
								"__typename": "Show",
								"availableEpisodes": map[string]any{
									"sub": 500,
									"dub": 400,
									"raw": 500,
								},
							},
						},
					},
				},
			},
			wantShows:   2,
			wantHasMore: false, // 2 < 40 (limit)
			wantErr:     false,
		},
		{
			name:  "search with full page indicates has more",
			query: "anime",
			opts: provider.SearchOptions{
				TranslationType: provider.TranslationSub,
				Page:            1,
			},
			serverResponse: func() map[string]any {
				edges := make([]map[string]any, 40)
				for i := 0; i < 40; i++ {
					edges[i] = map[string]any{
						"_id":        "show-id",
						"name":       "Anime",
						"__typename": "Show",
						"availableEpisodes": map[string]any{
							"sub": 12,
							"dub": 12,
							"raw": 12,
						},
					}
				}
				return map[string]any{
					"data": map[string]any{
						"shows": map[string]any{
							"edges": edges,
						},
					},
				}
			}(),
			wantShows:   40,
			wantHasMore: true, // 40 == 40 (limit)
			wantErr:     false,
		},
		{
			name:  "empty search results",
			query: "xyznonexistent",
			opts: provider.SearchOptions{
				TranslationType: provider.TranslationSub,
				Page:            1,
			},
			serverResponse: map[string]any{
				"data": map[string]any{
					"shows": map[string]any{
						"edges": []map[string]any{},
					},
				},
			},
			wantShows:   0,
			wantHasMore: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockServer(t, tt.serverResponse)
			defer server.Close()

			svc := newTestService(server.URL, testConfig())
			result, err := svc.SearchAnime(context.Background(), tt.query, tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("SearchAnime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if len(result.Shows) != tt.wantShows {
				t.Errorf("SearchAnime() got %d shows, want %d", len(result.Shows), tt.wantShows)
			}

			if result.HasMore != tt.wantHasMore {
				t.Errorf("SearchAnime() HasMore = %v, want %v", result.HasMore, tt.wantHasMore)
			}

			if result.Page != tt.opts.Page {
				t.Errorf("SearchAnime() Page = %d, want %d", result.Page, tt.opts.Page)
			}
		})
	}
}

func TestSearchAnime_ShowConversion(t *testing.T) {
	serverResponse := map[string]any{
		"data": map[string]any{
			"shows": map[string]any{
				"edges": []map[string]any{
					{
						"_id":        "test-id-123",
						"name":       "Test Anime",
						"__typename": "Show",
						"availableEpisodes": map[string]any{
							"sub": 24,
							"dub": 12,
							"raw": 24,
						},
					},
				},
			},
		},
	}

	server := mockServer(t, serverResponse)
	defer server.Close()

	svc := newTestService(server.URL, testConfig())
	result, err := svc.SearchAnime(context.Background(), "test", provider.SearchOptions{
		TranslationType: provider.TranslationSub,
		Page:            1,
	})

	if err != nil {
		t.Fatalf("SearchAnime() unexpected error: %v", err)
	}

	if len(result.Shows) != 1 {
		t.Fatalf("expected 1 show, got %d", len(result.Shows))
	}

	show := result.Shows[0]

	if show.ID != "test-id-123" {
		t.Errorf("Show.ID = %q, want %q", show.ID, "test-id-123")
	}

	if show.Name != "Test Anime" {
		t.Errorf("Show.Name = %q, want %q", show.Name, "Test Anime")
	}

	if show.AvailableEpisodes.Sub != 24 {
		t.Errorf("Show.AvailableEpisodes.Sub = %d, want %d", show.AvailableEpisodes.Sub, 24)
	}

	if show.AvailableEpisodes.Dub != 12 {
		t.Errorf("Show.AvailableEpisodes.Dub = %d, want %d", show.AvailableEpisodes.Dub, 12)
	}

	if show.AvailableEpisodes.Raw != 24 {
		t.Errorf("Show.AvailableEpisodes.Raw = %d, want %d", show.AvailableEpisodes.Raw, 24)
	}
}

func TestSearchAnime_GraphQLError(t *testing.T) {
	serverResponse := map[string]any{
		"errors": []map[string]any{
			{
				"message": "Invalid query",
			},
		},
	}

	server := mockServer(t, serverResponse)
	defer server.Close()

	svc := newTestService(server.URL, testConfig())
	_, err := svc.SearchAnime(context.Background(), "test", provider.SearchOptions{
		TranslationType: provider.TranslationSub,
		Page:            1,
	})

	if err == nil {
		t.Error("SearchAnime() expected error for GraphQL error response, got nil")
	}
}

func TestGetEpisodeList(t *testing.T) {
	tests := []struct {
		name           string
		showID         string
		serverResponse map[string]any
		wantSub        int
		wantDub        int
		wantRaw        int
		wantErr        bool
	}{
		{
			name:   "successful episode list",
			showID: "show-123",
			serverResponse: map[string]any{
				"data": map[string]any{
					"show": map[string]any{
						"_id": "show-123",
						"availableEpisodesDetail": map[string]any{
							"sub": []string{"1", "2", "3", "4", "5"},
							"dub": []string{"1", "2", "3"},
							"raw": []string{"1", "2", "3", "4", "5"},
						},
					},
				},
			},
			wantSub: 5,
			wantDub: 3,
			wantRaw: 5,
			wantErr: false,
		},
		{
			name:   "show with no dub",
			showID: "show-456",
			serverResponse: map[string]any{
				"data": map[string]any{
					"show": map[string]any{
						"_id": "show-456",
						"availableEpisodesDetail": map[string]any{
							"sub": []string{"1", "2"},
							"dub": []string{},
							"raw": []string{"1", "2"},
						},
					},
				},
			},
			wantSub: 2,
			wantDub: 0,
			wantRaw: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockServer(t, tt.serverResponse)
			defer server.Close()

			svc := newTestService(server.URL, testConfig())
			result, err := svc.GetEpisodeList(context.Background(), tt.showID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetEpisodeList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if len(result.Sub) != tt.wantSub {
				t.Errorf("GetEpisodeList() Sub count = %d, want %d", len(result.Sub), tt.wantSub)
			}

			if len(result.Dub) != tt.wantDub {
				t.Errorf("GetEpisodeList() Dub count = %d, want %d", len(result.Dub), tt.wantDub)
			}

			if len(result.Raw) != tt.wantRaw {
				t.Errorf("GetEpisodeList() Raw count = %d, want %d", len(result.Raw), tt.wantRaw)
			}
		})
	}
}

func TestGetEpisodeList_GraphQLError(t *testing.T) {
	serverResponse := map[string]any{
		"errors": []map[string]any{
			{
				"message": "Invalid query",
			},
		},
	}

	server := mockServer(t, serverResponse)
	defer server.Close()

	svc := newTestService(server.URL, testConfig())
	_, err := svc.GetEpisodeList(context.Background(), "invalid-id")

	if err == nil {
		t.Error("GetEpisodeList() expected error for GraphQL error response, got nil")
	}
}

func TestGetEpisodeList_ShowNotFound(t *testing.T) {
	serverResponse := map[string]any{
		"data": map[string]any{
			"show": nil,
		},
	}

	server := mockServer(t, serverResponse)
	defer server.Close()

	svc := newTestService(server.URL, testConfig())
	_, err := svc.GetEpisodeList(context.Background(), "nonexistent-id")

	if err == nil {
		t.Error("GetEpisodeList() expected error for nil show, got nil")
	}
}

func TestGetEpisodeSources(t *testing.T) {
	tests := []struct {
		name            string
		showID          string
		translationType provider.TranslationType
		episode         string
		serverResponse  map[string]any
		wantSources     int
		wantErr         bool
	}{
		{
			name:            "successful episode sources",
			showID:          "show-123",
			translationType: provider.TranslationSub,
			episode:         "1",
			serverResponse: map[string]any{
				"data": map[string]any{
					"episode": map[string]any{
						"episodeString": "1",
						"sourceUrls": []map[string]any{
							{
								"sourceName": "Default",
								"sourceUrl":  "5d565b575c5d5c",
								"type":       "iframe",
								"priority":   1.0,
							},
							{
								"sourceName": "Luf-Mp4",
								"sourceUrl":  "5d565b575c5d5c",
								"type":       "player",
								"priority":   0.8,
							},
						},
					},
				},
			},
			wantSources: 2,
			wantErr:     false,
		},
		{
			name:            "episode with no sources",
			showID:          "show-456",
			translationType: provider.TranslationDub,
			episode:         "99",
			serverResponse: map[string]any{
				"data": map[string]any{
					"episode": map[string]any{
						"episodeString": "99",
						"sourceUrls":    []map[string]any{},
					},
				},
			},
			wantSources: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("GetEpisodeSources not implemented yet")
			server := mockServer(t, tt.serverResponse)
			defer server.Close()

			svc := newTestService(server.URL, testConfig())
			result, err := svc.GetEpisodeSources(context.Background(), tt.showID, tt.translationType, tt.episode)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetEpisodeSources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if len(result.Sources) != tt.wantSources {
				t.Errorf("GetEpisodeSources() sources count = %d, want %d", len(result.Sources), tt.wantSources)
			}

			if result.EpisodeString != tt.episode {
				t.Errorf("GetEpisodeSources() EpisodeString = %q, want %q", result.EpisodeString, tt.episode)
			}
		})
	}
}

func TestGetEpisodeSources_SourceConversion(t *testing.T) {
	t.Skip("GetEpisodeSources not implemented yet")
	serverResponse := map[string]any{
		"data": map[string]any{
			"episode": map[string]any{
				"episodeString": "1",
				"sourceUrls": []map[string]any{
					{
						"sourceName": "TestProvider",
						"sourceUrl":  "encoded-url",
						"type":       "player",
						"priority":   0.9,
					},
				},
			},
		},
	}

	server := mockServer(t, serverResponse)
	defer server.Close()

	svc := newTestService(server.URL, testConfig())
	result, err := svc.GetEpisodeSources(context.Background(), "show-123", provider.TranslationSub, "1")

	if err != nil {
		t.Fatalf("GetEpisodeSources() unexpected error: %v", err)
	}

	if len(result.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(result.Sources))
	}

	source := result.Sources[0]

	if source.Name != "TestProvider" {
		t.Errorf("Source.Name = %q, want %q", source.Name, "TestProvider")
	}

	if source.URL != "encoded-url" {
		t.Errorf("Source.URL = %q, want %q", source.URL, "encoded-url")
	}

	if source.Type != "player" {
		t.Errorf("Source.Type = %q, want %q", source.Type, "player")
	}

	if source.Priority != 0.9 {
		t.Errorf("Source.Priority = %f, want %f", source.Priority, 0.9)
	}
}
