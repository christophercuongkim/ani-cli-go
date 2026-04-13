# AllAnime API Reference

Reverse-engineered from [ani-cli](https://github.com/pystardust/ani-cli).

## Endpoint & Headers

```
POST https://api.allanime.day/api
Content-Type: application/json
Referer: https://allmanga.to
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0
```

## GraphQL Queries

### 1. Search Anime

```graphql
query($search: SearchInput $limit: Int $page: Int $translationType: VaildTranslationTypeEnumType $countryOrigin: VaildCountryOriginEnumType) {
  shows(search: $search limit: $limit page: $page translationType: $translationType countryOrigin: $countryOrigin) {
    edges {
      _id
      name
      availableEpisodes
      __typename
    }
  }
}
```

**Variables:**
```json
{
  "search": {
    "allowAdult": false,
    "allowUnknown": false,
    "query": "search term here"
  },
  "limit": 40,
  "page": 1,
  "translationType": "sub",
  "countryOrigin": "ALL"
}
```

**Response structure:**
```json
{
  "data": {
    "shows": {
      "edges": [
        {
          "_id": "unique-show-id",
          "name": "Anime Title",
          "availableEpisodes": {
            "sub": 12,
            "dub": 10,
            "raw": 12
          },
          "__typename": "Show"
        }
      ]
    }
  }
}
```

### 2. Get Episode List

```graphql
query($showId: String!) {
  show(_id: $showId) {
    _id
    availableEpisodesDetail
  }
}
```

**Variables:**
```json
{
  "showId": "unique-show-id"
}
```

**Response structure:**
```json
{
  "data": {
    "show": {
      "_id": "unique-show-id",
      "availableEpisodesDetail": {
        "sub": ["1", "2", "3", "4", "5"],
        "dub": ["1", "2", "3"],
        "raw": ["1", "2", "3", "4", "5"]
      }
    }
  }
}
```

### 3. Get Episode Sources

```graphql
query($showId: String! $translationType: VaildTranslationTypeEnumType! $episodeString: String!) {
  episode(showId: $showId translationType: $translationType episodeString: $episodeString) {
    episodeString
    sourceUrls
  }
}
```

**Variables:**
```json
{
  "showId": "unique-show-id",
  "translationType": "sub",
  "episodeString": "1"
}
```

**Response structure:**
```json
{
  "data": {
    "episode": {
      "episodeString": "1",
      "sourceUrls": [
        {
          "sourceName": "Default",
          "sourceUrl": "--encoded-provider-id--",
          "type": "iframe",
          "priority": 1.0
        },
        {
          "sourceName": "Luf-Mp4",
          "sourceUrl": "--encoded-provider-id--",
          "type": "player",
          "priority": 0.8
        }
      ]
    }
  }
}
```

---

## Source URL Obfuscation

The `sourceUrl` field contains an **obfuscated provider ID**, not a direct URL. It must be decoded before use.

### The Algorithm: XOR 0x38

The encoding is simple: each byte of the original string is XOR'd with `0x38` (decimal 56), then represented as a 2-character lowercase hex string.

```
encode(char) = lowercase_hex(byte(char) XOR 0x38)
decode(hex)  = char(parse_hex(hex) XOR 0x38)
```

### Why XOR 0x38?

This shifts ASCII values in a reversible way:
- Encoding: `'A'` (0x41) XOR 0x38 = 0x79 → `"79"`
- Decoding: 0x79 XOR 0x38 = 0x41 → `'A'`

### Go Implementation

```go
// DecodeSourceURL decodes an obfuscated AllAnime source URL.
// The encoding is: hex(byte XOR 0x38) for each character.
func DecodeSourceURL(encoded string) (string, error) {
    if len(encoded)%2 != 0 {
        return "", fmt.Errorf("invalid encoded URL: odd length")
    }

    result := make([]byte, len(encoded)/2)
    for i := 0; i < len(encoded); i += 2 {
        b, err := strconv.ParseUint(encoded[i:i+2], 16, 8)
        if err != nil {
            return "", fmt.Errorf("invalid hex at position %d: %w", i, err)
        }
        result[i/2] = byte(b) ^ 0x38
    }
    return string(result), nil
}

// EncodeSourceURL encodes a string using AllAnime's obfuscation.
// Useful for testing.
func EncodeSourceURL(plain string) string {
    var result strings.Builder
    for i := 0; i < len(plain); i++ {
        result.WriteString(fmt.Sprintf("%02x", plain[i]^0x38))
    }
    return result.String()
}
```

### Complete Lookup Table (for reference)

If you prefer a lookup table approach, here's the full mapping:

| Hex | Char | Hex | Char | Hex | Char | Hex | Char |
|-----|------|-----|------|-----|------|-----|------|
| 79 | A | 7a | B | 7b | C | 7c | D |
| 7d | E | 7e | F | 7f | G | 70 | H |
| 71 | I | 72 | J | 73 | K | 74 | L |
| 75 | M | 76 | N | 77 | O | 68 | P |
| 69 | Q | 6a | R | 6b | S | 6c | T |
| 6d | U | 6e | V | 6f | W | 60 | X |
| 61 | Y | 62 | Z | | | | |
| 59 | a | 5a | b | 5b | c | 5c | d |
| 5d | e | 5e | f | 5f | g | 50 | h |
| 51 | i | 52 | j | 53 | k | 54 | l |
| 55 | m | 56 | n | 57 | o | 48 | p |
| 49 | q | 4a | r | 4b | s | 4c | t |
| 4d | u | 4e | v | 4f | w | 40 | x |
| 41 | y | 42 | z | | | | |
| 08 | 0 | 09 | 1 | 0a | 2 | 0b | 3 |
| 0c | 4 | 0d | 5 | 0e | 6 | 0f | 7 |
| 00 | 8 | 01 | 9 | | | | |
| 02 | : | 17 | / | 16 | . | 15 | - |
| 67 | _ | 46 | ~ | 07 | ? | 1b | # |
| 63 | [ | 65 | ] | 78 | @ | 19 | ! |
| 1c | $ | 1e | & | 10 | ( | 11 | ) |
| 12 | * | 13 | + | 14 | , | 03 | ; |
| 05 | = | 1d | % | | | | |

### Example

```
Encoded: "50515b5d5e17175d405955485d1654555d"
         ↓ XOR each byte with 0x38
Decoded: "hicef//examplel.com"
```

After decoding, append `/clock.json` to get the provider's stream info endpoint.

---

## Provider Resolution

The decoded source URL is a **provider ID**, not the final video URL. To get playable streams:

1. Decode the `sourceUrl` using XOR 0x38
2. Append `/clock.json` to the decoded string
3. Fetch that URL to get stream metadata

### Supported Providers

| Provider | Format | Quality |
|----------|--------|---------|
| wixmp (repackager.wixmp.com) | m3u8 | Multi-resolution |
| sharepoint | mp4 | Single file |
| youtube (Yt-mp4) | mp4 | Single file |
| hianime | m3u8 | Multi-resolution |

### Provider Response Handling

**m3u8 streams (wixmp, hianime):**
- Fetch the master.m3u8 playlist
- Parse available resolutions (1080p, 720p, 480p, etc.)
- Select based on user quality preference

**mp4 streams (sharepoint, youtube):**
- Direct URL to video file
- Single quality option

---

## Translation Types

The API uses `VaildTranslationTypeEnumType` (note the typo in "Valid"):

| Value | Description |
|-------|-------------|
| `sub` | Japanese audio with subtitles |
| `dub` | English dubbed audio |
| `raw` | Japanese audio, no subtitles |

---

## Error Handling

- **Rate limiting:** Implement exponential backoff
- **Mirror rotation:** If `api.allanime.day` fails, try alternate mirrors
- **Invalid responses:** Check for `errors` field in GraphQL response

---

## Notes

- The API is unofficial and may change without notice
- Always send the correct `Referer` header or requests will be blocked
- Source URLs change frequently; don't cache decoded URLs for long
