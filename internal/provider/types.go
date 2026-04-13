package provider

// TranslationType for sub/dub/raw selection
type TranslationType string

const (
	TranslationSub TranslationType = "sub"
	TranslationDub TranslationType = "dub"
	TranslationRaw TranslationType = "raw"
)

// SearchOptions for searching anime
type SearchOptions struct {
	TranslationType TranslationType
	Page            int
}

// SearchResult contains search results with pagination info
type SearchResult struct {
	Shows   []Show
	Page    int
	HasMore bool
}

// Show represents an anime series
type Show struct {
	ID                string
	Name              string
	AvailableEpisodes AvailableEpisodes
}

// AvailableEpisodes count per translation type
type AvailableEpisodes struct {
	Sub int
	Dub int
	Raw int
}

// EpisodeList contains available episode strings per translation type
type EpisodeList struct {
	Sub []string
	Dub []string
	Raw []string
}

// Episode with streaming sources
type Episode struct {
	EpisodeString string
	Sources       []Source
}

// Source contains provider info for streaming
type Source struct {
	Name     string
	URL      string
	Type     string
	Priority float64
}
