package allanime

import (
	"github.com/hasura/go-graphql-client"

	"github.com/christophercuongkim/ani-cli-go/internal/provider"
)

// CountryOrigin for filtering by country
type CountryOrigin string

const (
	CountryAll CountryOrigin = "ALL"
	CountryJP  CountryOrigin = "JP"
	CountryCN  CountryOrigin = "CN"
	CountryKR  CountryOrigin = "KR"
)

// SearchInput for search query variables
type SearchInput struct {
	AllowAdult   graphql.Boolean `json:"allowAdult"`
	AllowUnknown graphql.Boolean `json:"allowUnknown"`
	Query        graphql.String  `json:"query"`
}

// AvailableEpisodes count per translation type
type AvailableEpisodes struct {
	Sub graphql.Int `graphql:"sub"`
	Dub graphql.Int `graphql:"dub"`
	Raw graphql.Int `graphql:"raw"`
}

// Show edge in search results
type Show struct {
	ID                graphql.String    `graphql:"_id"`
	Name              graphql.String    `graphql:"name"`
	AvailableEpisodes AvailableEpisodes `graphql:"availableEpisodes"`
	Typename          graphql.String    `graphql:"__typename"`
}

// SearchQuery for searching anime
// Query: shows(search: $search, limit: $limit, page: $page, translationType: $translationType, countryOrigin: $countryOrigin)
type SearchQuery struct {
	Shows struct {
		Edges []Show `graphql:"edges"`
	} `graphql:"shows(search: $search, limit: $limit, page: $page, translationType: $translationType, countryOrigin: $countryOrigin)"`
}

// SearchVariables for search query
type SearchVariables struct {
	Search          SearchInput              `json:"search"`
	Limit           graphql.Int              `json:"limit"`
	Page            graphql.Int              `json:"page"`
	TranslationType provider.TranslationType `json:"translationType"`
	CountryOrigin   CountryOrigin            `json:"countryOrigin"`
}

// ToMap converts SearchVariables to map for graphql client
func (v SearchVariables) ToMap() map[string]any {
	return map[string]any{
		"search":          v.Search,
		"limit":           v.Limit,
		"page":            v.Page,
		"translationType": v.TranslationType,
		"countryOrigin":   v.CountryOrigin,
	}
}

// AvailableEpisodesDetail lists episode strings per translation type
type AvailableEpisodesDetail struct {
	Sub []graphql.String `graphql:"sub"`
	Dub []graphql.String `graphql:"dub"`
	Raw []graphql.String `graphql:"raw"`
}

// ShowDetail for episode list query
type ShowDetail struct {
	ID                      graphql.String          `graphql:"_id"`
	AvailableEpisodesDetail AvailableEpisodesDetail `graphql:"availableEpisodesDetail"`
}

// EpisodeListQuery for getting available episodes
// Query: show(_id: $showId)
type EpisodeListQuery struct {
	Show ShowDetail `graphql:"show(_id: $showId)"`
}

// EpisodeListVariables for episode list query
type EpisodeListVariables struct {
	ShowID graphql.String `json:"showId"`
}

// ToMap converts EpisodeListVariables to map for graphql client
func (v EpisodeListVariables) ToMap() map[string]any {
	return map[string]any{
		"showId": v.ShowID,
	}
}

// SourceURL contains provider info for episode
type SourceURL struct {
	SourceName graphql.String  `graphql:"sourceName"`
	SourceURL  graphql.String  `graphql:"sourceUrl"`
	Type       graphql.String  `graphql:"type"`
	Priority   graphql.Float   `graphql:"priority"`
}

// Episode with source URLs
type Episode struct {
	EpisodeString graphql.String `graphql:"episodeString"`
	SourceUrls    []SourceURL    `graphql:"sourceUrls"`
}

// EpisodeSourcesQuery for getting episode streaming sources
// Query: episode(showId: $showId, translationType: $translationType, episodeString: $episodeString)
type EpisodeSourcesQuery struct {
	Episode Episode `graphql:"episode(showId: $showId, translationType: $translationType, episodeString: $episodeString)"`
}

// EpisodeSourcesVariables for episode sources query
type EpisodeSourcesVariables struct {
	ShowID          graphql.String           `json:"showId"`
	TranslationType provider.TranslationType `json:"translationType"`
	EpisodeString   graphql.String           `json:"episodeString"`
}

// ToMap converts EpisodeSourcesVariables to map for graphql client
func (v EpisodeSourcesVariables) ToMap() map[string]any {
	return map[string]any{
		"showId":          v.ShowID,
		"translationType": v.TranslationType,
		"episodeString":   v.EpisodeString,
	}
}
