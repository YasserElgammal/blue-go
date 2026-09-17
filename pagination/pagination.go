// Package pagination contains database-agnostic pagination values.
package pagination

// Pagination describes a page of records.
type Pagination struct {
	Page    int
	PerPage int
	Total   int
}

// Metadata is the normalized pagination information returned to API clients.
type Metadata struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}

// Metadata normalizes the pagination values and calculates the final page.
func (p Pagination) Metadata() Metadata {
	page := max(p.Page, 1)
	perPage := max(p.PerPage, 1)
	total := max(p.Total, 0)
	lastPage := 0
	if total > 0 {
		lastPage = total / perPage
		if total%perPage != 0 {
			lastPage++
		}
	}
	return Metadata{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		LastPage:    lastPage,
	}
}
