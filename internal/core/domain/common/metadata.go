package domain

import "math"

type Metadata struct {
	CurrentPage  int `json:"currentPage,omitempty"`
	PageSize     int `json:"pageSize,omitempty"`
	FirstPage    int `json:"firstPage,omitempty"`
	LastPage     int `json:"lastPage,omitempty"`
	TotalRecords int `json:"totalRecords,omitempty"`
}

func CalculateMetadata(totalRecords int, offset, limit int) Metadata {
	if totalRecords == 0 {
		return Metadata{
			CurrentPage:  (offset / limit) + 1,
			PageSize:     limit,
			FirstPage:    1,
			LastPage:     1,
			TotalRecords: 0,
		}
	}
	return Metadata{
		CurrentPage:  (offset / limit) + 1,
		PageSize:     limit,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(limit))),
		TotalRecords: totalRecords,
	}
}

type PaginationMetadata struct {
	TotalRecords int64  `json:"totalRecords"`
	Page         int    `json:"page,omitempty"`
	Limit        int    `json:"limit"`
	TotalPages   int    `json:"totalPages,omitempty"`
	HasMore      bool   `json:"hasMore,omitempty"`
	NextCursor   string `json:"nextCursor,omitempty"`
}