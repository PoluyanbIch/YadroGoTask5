package core

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	concurrency int
	status      atomic.Value
	isUpdating  int32
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	s := &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		concurrency: concurrency,
		isUpdating:  0,
	}
	s.status.Store(StatusIdle)
	return s, nil
}

func (s *Service) Update(ctx context.Context) (err error) {
	if !atomic.CompareAndSwapInt32(&s.isUpdating, 0, 1) {
		return ErrUpdateInProgress
	}
	defer atomic.StoreInt32(&s.isUpdating, 0)
	s.status.Store(StatusRunning)

	defer func() { s.status.Store(StatusIdle) }()
	var ids []int
	existIDs, err := s.db.IDs(ctx)
	if err != nil {
		return err
	}
	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return err
	}
	for i := range lastID + 1 {
		if !slices.Contains(existIDs, i) && i != 404 && i != 0 {
			ids = append(ids, i)
		}
	}
	for i := 0; i < len(ids); i += s.concurrency {
		end := i + s.concurrency
		if end > len(ids) {
			end = len(ids)
		}
		var wg sync.WaitGroup
		comicsChan := make(chan Comics, s.concurrency)
		for _, id := range ids[i:end] {
			wg.Go(func() {
				xkcdComics, err := s.xkcd.Get(ctx, id)
				if err != nil {
					s.log.Error("xkcd.get error", "error", err)
					return
				}
				words, err := s.words.Norm(ctx, xkcdComics.Title+" "+xkcdComics.Description)
				if err != nil {
					s.log.Error("words.norm error", "error", err)
					return
				}
				wordsMap := make(map[string]int)
				for _, w := range words {
					wordsMap[w]++
				}
				comics := Comics{
					ID:    xkcdComics.ID,
					URL:   xkcdComics.URL,
					Words: wordsMap,
				}

				comicsChan <- comics
			})
		}
		go func() {
			wg.Wait()
			close(comicsChan)
		}()
		for c := range comicsChan {
			if err := s.db.Add(ctx, c); err != nil {
				s.log.Error("db.add error", "error", err)
			}
		}
	}
	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		s.log.Error("db.stats", "error", err)
		return ServiceStats{}, err
	}
	comicsTotal, err := s.xkcd.LastID(ctx)
	if err != nil {
		s.log.Error("xkcd.lastid", "error", err)
		return ServiceStats{}, err
	}
	return ServiceStats{
		DBStats:     dbStats,
		ComicsTotal: comicsTotal - 1,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	return s.status.Load().(ServiceStatus)
}

func (s *Service) Drop(ctx context.Context) error {
	if err := s.db.Drop(ctx); err != nil {
		s.log.Error("db.drop", "error", err)
		return err
	}
	return nil
}
