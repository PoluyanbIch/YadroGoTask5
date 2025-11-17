package core

import (
	"context"
)

type Searcher interface {
	Search(context.Context, string, int) ([]Comic, error)
}

type Update interface {
	Update(context.Context) error
}

type DB interface {
	Read(context.Context) ([]DBComic, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}
