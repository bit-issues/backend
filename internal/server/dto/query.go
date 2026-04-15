package dto

// PaginationQuery represents pagination query parameters.
type PaginationQuery struct {
	Limit  int `query:"limit"  validate:"omitempty,min=1,max=100" default:"20"`
	Offset int `query:"offset" validate:"omitempty,min=0"         default:"0"`
}

// SortQuery represents sorting query parameters.
type SortQuery struct {
	Sort string `query:"sort" validate:"omitempty"`
}
