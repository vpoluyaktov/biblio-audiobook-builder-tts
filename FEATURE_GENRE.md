# Feature Spec: Dynamic Genre Metadata for M4B Audiobooks

## Problem Statement

In `internal/server/worker.go` line 843, the M4B genre metadata is hardcoded:

```go
Genre: "Audiobook",
```

This means every audiobook produced by the system carries `"Audiobook"` as its genre tag, discarding actual genre information that is available from both the OPDS catalog feed and the ebook file itself.

---

## Current State Analysis

### Where genre data already exists

1. **OPDS feed entries** -- The `CatalogEntry` struct (`internal/opds/common.go:24`) already has a `Categories []string` field. Both OPDS 1.x and OPDS 2.0 parsers populate it:
   - OPDS 1.x: extracted from `<category term="..." label="...">` elements in `convertOPDS1Entry` (`internal/opds/opds1.go:200-206`).
   - OPDS 2.0: extracted from `subject` field (string, array of strings, or array of objects) in `convertOPDS2Publication` (`internal/opds/opds2.go:242-257`).

2. **Ebook parser library** (`biblio-ebook-parser`) -- The unified `parser.Metadata` struct has `Genres []string`:
   - EPUB: populated from `<dc:subject>` elements (`formats/epub/epub.go:125`).
   - FB2: populated from `<genre>` elements (`formats/fb2/fb2.go:153`).

3. **Plaintext renderer** -- The `plaintext.Book.Metadata` map (`map[string]string`) is populated by the renderer but does **not** currently include genres. The renderer only copies `description` into the metadata map (`renderer/plaintext/plaintext.go:71-73`). Genres are present on the upstream `parser.Book.Metadata.Genres` but are lost during the `plaintext.RenderContent` step.

### Where genre data is NOT carried through

The following structs and data paths **lack** genre information today:

| Layer | Struct | File | Gap |
|-------|--------|------|-----|
| Internal book model | `parser.Book` (local) | `internal/parser/parser.go` | No `Genre` / `Genres` field |
| Parser adapter | `ParseBook`, `ParseFile`, `ParseReader` | `internal/parser/adapter.go` | Drops `Genres` from upstream `parser.Metadata` during conversion |
| Preview | `Preview` | `internal/server/preview.go` | No genre/categories field |
| Job | `Job` / `JobDTO` / `storage.Job` | `internal/server/job.go`, `internal/storage/db.go` | No genre field |
| OPDS download request | inline struct in `handleOPDSDownload` | `internal/server/opds_handlers.go:442-449` | No `categories` field in request body |

### Data flow summary (current)

```
OPDS Feed
  -> CatalogEntry.Categories []string  (POPULATED)
  -> handleOPDSDownload request         (LOST -- categories not sent from frontend)
  -> parser.ParseReader(data, format)
    -> biblio-ebook-parser Metadata.Genres []string  (POPULATED in upstream parser)
    -> plaintext.RenderContent
      -> plaintext.Book.Metadata map    (LOST -- genres not copied into map)
    -> internal parser.Book             (LOST -- no Genre field)
  -> Preview                            (LOST -- no genre field)
  -> NewJob(...)                        (LOST -- no genre parameter)
  -> worker.buildM4B()
    -> M4BOptions.Genre = "Audiobook"   (HARDCODED)
```

---

## Proposed Changes

### Design Decision: Genre Resolution Order

The genre for the M4B file should be resolved with the following priority:

1. **OPDS categories** (from the catalog entry the user selected) -- these are curated library classifications.
2. **Ebook file genres** (from the parsed ebook metadata) -- these come from the file's embedded metadata (e.g., `dc:subject` in EPUB, `<genre>` in FB2).
3. **Fallback to `"Audiobook"`** -- only if neither source provides a genre.

### Design Decision: Multiple Genres

When multiple genres/categories are available:
- **Join them with `, ` (comma-space)** into a single string for the M4B `Genre` tag.
- M4B (MP4 container) metadata only supports a single genre string field. Comma-separated values are the de facto standard used by iTunes, Audiobookshelf, and other audiobook managers.
- **Limit to first 5 genres** to avoid excessively long metadata strings. Truncate with no ellipsis.

### Design Decision: Genre Sanitization

- Trim leading/trailing whitespace from each genre.
- Remove empty strings from the list after trimming.
- Remove exact duplicates (case-insensitive dedup, preserve original casing of first occurrence).
- No further character filtering -- genres may legitimately contain characters like `&`, `/`, `(`, etc.

---

## Struct Changes

### 1. `internal/parser/parser.go` -- Add `Genre` field to `Book`

```go
type Book struct {
    Title          string
    Author         string
    Series         string
    SeriesNumber   string
    Description    string
    Genre          string            // NEW: comma-separated genre string
    Chapters       []Chapter
    CoverImage     []byte
    CoverImageName string
    CoverImageType string
    Metadata       map[string]string
}
```

**Rationale:** A single `Genre string` (not `[]string`) because this struct is consumed by the M4B builder which needs a single string. The joining and dedup logic happens at population time, not at consumption time.

### 2. `internal/server/preview.go` -- Add `Genre` field to `Preview`

```go
type Preview struct {
    // ... existing fields ...
    Genre string `json:"genre,omitempty"` // NEW: genre from ebook metadata
    // ... rest of fields ...
}
```

Add this field after `Description` (line 21).

### 3. `internal/server/job.go` -- Add `Genre` field to `Job` and `JobDTO`

In `Job` struct, add under the "Book metadata" section (after `BookAuthor`):

```go
BookGenre  string `json:"book_genre,omitempty"` // NEW
```

In `JobDTO` struct, add in the same position:

```go
BookGenre  string `json:"book_genre,omitempty"` // NEW
```

### 4. `internal/storage/db.go` -- Add `Genre` field to `storage.Job`

```go
BookGenre string `json:"book_genre"` // NEW
```

**Note:** This also requires a schema migration to add a `book_genre TEXT DEFAULT ''` column to the `jobs` table. Check the existing migration pattern in `db.go` for how columns are added.

### 5. `internal/server/opds_handlers.go` -- Add `Categories` to download request

In `handleOPDSDownload`, the inline request struct (line 442-449) needs:

```go
var req struct {
    URL        string   `json:"url"`
    Title      string   `json:"title"`
    Format     string   `json:"format"`
    Author     string   `json:"author"`
    SourceID   string   `json:"source_id"`
    CoverURL   string   `json:"cover_url"`
    Categories []string `json:"categories"` // NEW
}
```

---

## Data Flow Changes (File by File)

### File 1: `internal/parser/adapter.go`

**All three functions** (`ParseBook`, `ParseFile`, `ParseReader`) must extract genres from the upstream parser and set them on the local `Book.Genre` field.

In `ParseBook` (line 56-64), after setting `Metadata`:
```go
// After: Metadata: plaintextBook.Metadata,
// Add genre from upstream parser metadata
if len(book.Metadata.Genres) > 0 {
    result.Genre = joinGenres(book.Metadata.Genres)
}
```

In `ParseFile` (line 146-153), after setting `Series`:
```go
// Set genre from upstream metadata
if len(unifiedBook.Metadata.Genres) > 0 {
    book.Genre = joinGenres(unifiedBook.Metadata.Genres)
}
```

In `ParseReader` (line 224-230), same pattern as `ParseFile`.

Add a helper function in the same file:
```go
// joinGenres deduplicates, trims, and joins genre strings.
// Returns at most 5 genres joined by ", ".
func joinGenres(genres []string) string { ... }
```

This function must:
1. Trim whitespace from each entry.
2. Remove empty strings.
3. Deduplicate case-insensitively (keep first occurrence's casing).
4. Take at most 5 entries.
5. Join with `", "`.

### File 2: `internal/server/preview.go`

In `CreatePreview` (line 302-317), after setting `Description`:
```go
preview.Genre = book.Genre
```

### File 3: `internal/server/opds_handlers.go`

In `handleOPDSDownload` (around line 524-528, after `preview.FilePath = tempPath`):

If the OPDS download request includes `Categories` and the preview's genre is empty (or even if it is not -- OPDS categories take priority per design decision):
```go
if len(req.Categories) > 0 {
    preview.Genre = joinOPDSCategories(req.Categories)
}
```

Add a helper (or reuse the parser's `joinGenres` by extracting it to a shared utility):
```go
func joinOPDSCategories(categories []string) string { ... }
```

Same logic as `joinGenres`: trim, dedup, limit 5, join with `", "`.

**Important:** The OPDS categories override the ebook-parsed genre because OPDS data represents the library's curated classification, which is typically more accurate than embedded ebook metadata.

### File 4: `internal/server/job.go`

In `NewJob` function -- add a `genre` parameter:

```go
func NewJob(fileName, filePath, provider, voice, language, genre string, speed, pitch float64, useSentencePauses bool) *Job {
```

Set `BookGenre: genre` in the returned struct.

In `SetBook` method (line 218-227), also copy genre from the parsed book if the job's genre is still empty (fallback to ebook metadata):
```go
func (j *Job) SetBook(book *parser.Book) {
    j.mu.Lock()
    defer j.mu.Unlock()
    j.book = book
    if book != nil {
        j.BookTitle = book.Title
        j.BookAuthor = book.Author
        j.TotalChapters = len(book.Chapters)
        // Fill genre from parsed book if not already set by OPDS
        if j.BookGenre == "" && book.Genre != "" {
            j.BookGenre = book.Genre
        }
    }
}
```

In `Clone` method (line 265-309), copy `BookGenre`:
```go
BookGenre: j.BookGenre,
```

### File 5: `internal/server/server.go`

Update all `NewJob` call sites to pass genre:

- Line 741 (upload handler): Pass `""` as genre (uploads do not carry OPDS categories; genre will be populated later via `SetBook` from the parsed ebook).
- `storageJobToJob` (line 167-192): Map `BookGenre` between `storage.Job` and `server.Job`.
- `jobToStorageJob` (line 116-165): Map `BookGenre` in the reverse direction.

### File 6: `internal/server/opds_handlers.go`

Update the `NewJob` call at line 623:
```go
job := NewJob(preview.FileName, preview.FilePath, provider, voice, language, preview.Genre, speed, pitch, useSentencePauses)
```

### File 7: `internal/storage/db.go`

- Add `BookGenre` to the `Job` struct.
- Add `book_genre` column to the `jobs` table schema (migration).
- Update all SQL queries that read/write the `jobs` table to include the new column.

### File 8: `internal/server/worker.go`

**The key change.** At line 837-846, replace the hardcoded genre:

```go
// Resolve genre with fallback
genre := job.BookGenre
if genre == "" && book.Genre != "" {
    genre = book.Genre
}
if genre == "" {
    genre = "Audiobook"
}

options := audio.M4BOptions{
    Title:           book.Title,
    Author:          book.Author,
    Album:           book.Title,
    Series:          book.Series,
    SeriesNumber:    book.SeriesNumber,
    Genre:           genre,  // was: "Audiobook"
    Description:     book.Description,
    GapBetweenChaps: gapDuration,
}
```

The three-tier fallback is:
1. `job.BookGenre` -- set from OPDS categories (via preview) or from a previous `SetBook` call.
2. `book.Genre` -- set from ebook file metadata during parsing.
3. `"Audiobook"` -- hardcoded default.

### File 9: Frontend JavaScript (OPDS UI)

The frontend must send `categories` in the OPDS download request body. When the user clicks "Download" on an OPDS catalog entry, the JavaScript already has access to the `CatalogEntry.categories` array from the browse/search API response.

In the download fetch call, add:
```javascript
categories: entry.categories || []
```

This is a Frontend Developer task. The exact file depends on the JS location (likely `internal/server/assets/` or embedded templates).

---

## Edge Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| OPDS entry has categories, ebook file also has genres | OPDS categories win (higher priority). |
| OPDS entry has empty categories, ebook has genres | Ebook genres are used. |
| Neither OPDS nor ebook has genre data | Falls back to `"Audiobook"`. |
| OPDS categories contain duplicates (e.g., `["Fiction", "fiction", "Fiction"]`) | Deduplicated to `"Fiction"`. |
| Genre string contains special characters (`"Science Fiction & Fantasy"`) | Preserved as-is. No escaping needed for M4B metadata. |
| Categories array has > 5 entries | Truncated to first 5 entries. |
| Categories contain empty strings or whitespace-only entries | Filtered out before joining. |
| Single genre value | Used directly (no trailing comma). |
| Genre value is extremely long (> 255 chars after joining) | No explicit truncation. M4B/MP4 metadata has no hard limit on genre length. If this becomes a problem in practice, add a 255-char limit later. |
| Direct file upload (not OPDS) | Genre comes only from ebook parser. Falls back to `"Audiobook"` if ebook has no genre metadata. |
| Job resumed from database (worker restart) | `BookGenre` is persisted in the `jobs` table and restored via `storageJobToJob`. The ebook is re-parsed during `processJob`, so `book.Genre` is also available as secondary source. |

---

## File-by-File Change Summary for Backend Developer

| # | File | Change Type | Description |
|---|------|-------------|-------------|
| 1 | `internal/parser/parser.go` | Add field | Add `Genre string` to `Book` struct |
| 2 | `internal/parser/adapter.go` | Add logic | Extract genres from upstream parser in all 3 parse functions; add `joinGenres` helper |
| 3 | `internal/server/preview.go` | Add field + logic | Add `Genre` to `Preview` struct; populate in `CreatePreview` |
| 4 | `internal/server/job.go` | Add field + update | Add `BookGenre` to `Job` and `JobDTO`; update `NewJob` signature; update `SetBook` for fallback; update `Clone` |
| 5 | `internal/server/server.go` | Update mappings | Update `NewJob` call in upload handler; update `jobToStorageJob` and `storageJobToJob` |
| 6 | `internal/server/opds_handlers.go` | Add field + logic | Add `Categories` to download request struct; pass `preview.Genre` to `NewJob`; set genre on preview from OPDS categories |
| 7 | `internal/storage/db.go` | Add field + migration | Add `BookGenre` to `storage.Job`; add DB column; update SQL queries |
| 8 | `internal/server/worker.go` | Core change | Replace hardcoded `"Audiobook"` with 3-tier genre resolution |

## File-by-File Change Summary for Frontend Developer

| # | File | Change Type | Description |
|---|------|-------------|-------------|
| 1 | OPDS UI JavaScript | Update fetch | Include `categories` array in the OPDS download POST request body |

---

## Test Cases for QA Engineer

### Unit Tests: `internal/parser/adapter_test.go`

| # | Test Case | Input | Expected Output |
|---|-----------|-------|-----------------|
| 1 | Genres extracted from EPUB | EPUB file with `<dc:subject>Fiction</dc:subject><dc:subject>Adventure</dc:subject>` | `book.Genre == "Fiction, Adventure"` |
| 2 | Genres extracted from FB2 | FB2 file with `<genre>sf</genre><genre>detective</genre>` | `book.Genre == "sf, detective"` |
| 3 | No genres in ebook | EPUB with no `<dc:subject>` elements | `book.Genre == ""` |
| 4 | Duplicate genres | Genres `["Fiction", "fiction", "Fantasy", "Fiction"]` | `"Fiction, Fantasy"` |
| 5 | Whitespace-only genre entries | Genres `["Fiction", "  ", "", "Adventure"]` | `"Fiction, Adventure"` |
| 6 | More than 5 genres | 8 genres | Only first 5 joined |
| 7 | Single genre | `["Audiobook"]` | `"Audiobook"` (no comma) |

### Unit Tests: `joinGenres` helper

| # | Test Case | Input | Expected Output |
|---|-----------|-------|-----------------|
| 8 | Empty slice | `[]string{}` | `""` |
| 9 | All empty strings | `[]string{"", "  ", ""}` | `""` |
| 10 | Special characters | `[]string{"Science Fiction & Fantasy"}` | `"Science Fiction & Fantasy"` |
| 11 | Exactly 5 genres | 5 genres | All 5 joined |
| 12 | Case-insensitive dedup | `["Sci-Fi", "sci-fi", "SCI-FI"]` | `"Sci-Fi"` |

### Unit Tests: `internal/server/worker_test.go`

| # | Test Case | Genre Resolution |
|---|-----------|-----------------|
| 13 | Job has BookGenre, book has Genre | `job.BookGenre` used |
| 14 | Job has empty BookGenre, book has Genre | `book.Genre` used |
| 15 | Both empty | `"Audiobook"` used |
| 16 | Job has BookGenre, book Genre is empty | `job.BookGenre` used |

### Unit Tests: `internal/server/job_test.go`

| # | Test Case | Description |
|---|-----------|-------------|
| 17 | SetBook with genre when BookGenre empty | BookGenre populated from book.Genre |
| 18 | SetBook with genre when BookGenre already set | BookGenre NOT overwritten |
| 19 | Clone includes BookGenre | Verify Clone() copies BookGenre to JobDTO |

### Integration Tests: `internal/server/opds_handlers_test.go`

| # | Test Case | Description |
|---|-----------|-------------|
| 20 | OPDS download with categories | Categories in request body populate preview.Genre |
| 21 | OPDS download without categories | preview.Genre comes from ebook parse |
| 22 | OPDS convert passes genre to job | Job created from preview carries genre |

### Unit Tests: `internal/server/preview_test.go`

| # | Test Case | Description |
|---|-----------|-------------|
| 23 | CreatePreview with genre | Book with Genre populates Preview.Genre |
| 24 | CreatePreview without genre | Preview.Genre is empty string |

---

## Migration Notes

- The `book_genre` column in SQLite should use `TEXT DEFAULT ''` to avoid breaking existing rows.
- Existing jobs in the database will have an empty `book_genre`, which is correct -- they will fall back to `"Audiobook"` if re-processed.
- No data backfill is needed.

## Risk Assessment

- **Low risk:** All changes are additive. The fallback to `"Audiobook"` preserves current behavior for any path that does not populate genre data.
- **Frontend coordination required:** The OPDS download request body change requires frontend and backend to agree on the `categories` field name and type (`[]string`).
- **No breaking API changes:** The new `genre` and `book_genre` fields use `omitempty` JSON tags, so existing API consumers see no difference unless they opt in.
