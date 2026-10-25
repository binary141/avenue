package sdk

// SortDirection controls list ordering for endpoints that accept a `sort`
// query parameter (e.g. GET /v1/folder/list). Values are lowercase and are
// valid directly as SQL ORDER BY direction keywords (case-insensitive).
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// SortBy names the column a list is ordered by. The zero value ("") leaves
// the choice to the server. Which values each endpoint accepts is documented
// on that endpoint's method; anything it doesn't recognize falls back to the
// server default.
type SortBy string

const (
	// SortByName orders by name. Accepted by folder listings and the trash.
	SortByName SortBy = "name"
	// SortBySize orders by file size. Accepted by folder listings and the trash.
	SortBySize SortBy = "size"
	// SortByDate orders folder listings by creation time. Not accepted by the
	// trash, which always orders by deletion time unless name or size is given.
	SortByDate SortBy = "date"
)
