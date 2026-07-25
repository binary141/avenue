package sdk

// SortDirection controls list ordering for endpoints that accept a `sort`
// query parameter (e.g. GET /v1/folder/list). Values are lowercase and are
// valid directly as SQL ORDER BY direction keywords (case-insensitive).
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)
