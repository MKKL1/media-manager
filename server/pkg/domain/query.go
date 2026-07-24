package domain

type SortField string
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

const (
	SortByCreatedAt SortField = "created_at"
	SortByUpdatedAt SortField = "updated_at"
	SortByTitle     SortField = "title"
	SortByStatus    SortField = "status"
	SortByType      SortField = "type"
	SortByLastSync  SortField = "last_sync"
)

type Pagination struct {
	Limit  int
	Offset int
}

type MediaQuery struct {
	Type     MediaType
	SortBy   SortField
	SortDir  SortDirection
	Paginate Pagination
}

type MediaPage struct {
	Items  []MediaSummary
	Total  int
	Offset int
	Limit  int
}
