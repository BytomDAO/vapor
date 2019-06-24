package api

type PaginationQuery struct {
	Start uint64 `json:"start"`
	Limit uint64 `json:"limit"`
}
