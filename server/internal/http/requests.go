package http

import (
	"net/http"
	"server/internal/domain"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/gorilla/schema"
)

var (
	validate = validator.New()
	decoder  = schema.NewDecoder()
)

func init() {
	decoder.SetAliasTag("url")
	decoder.IgnoreUnknownKeys(true)
}

func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return domain.ErrInvalidInput
	}
	return validate.Struct(dst)
}

func decodeQuery(r *http.Request, dst any) error {
	if err := decoder.Decode(dst, r.URL.Query()); err != nil {
		return domain.ErrInvalidInput
	}
	return validate.Struct(dst)
}

type pullMediaRequest struct {
	Provider  string           `json:"provider"   validate:"required"`
	ID        string           `json:"id"         validate:"required"`
	MediaType domain.MediaType `json:"media_type" validate:"required,oneof=movie tv"`
}

type queryMediaRequest struct {
	Type   string `url:"type"   validate:"omitempty,oneof=movie tv"`
	SortBy string `url:"sort"   validate:"omitempty"`
	Offset int    `url:"offset" validate:"min=0"`
	Limit  int    `url:"limit"  validate:"min=0,max=100"`
}

func (q queryMediaRequest) ToDomain() domain.MediaQuery {
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	return domain.MediaQuery{
		Type:    domain.MediaType(q.Type),
		SortBy:  domain.SortField(q.SortBy),
		SortDir: "DESC",
		Paginate: domain.Pagination{
			Limit:  limit,
			Offset: q.Offset,
		},
	}
}
