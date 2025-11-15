package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"yadro.com/course/api/config"
	"yadro.com/course/api/core"
)

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		replies := make(map[string]string)

		for name, pinger := range pingers {
			err := pinger.Ping(ctx)
			if err != nil {
				log.Warn("ping failed", "service", name, "error", err)
				replies[name] = "unavailable"
			} else {
				replies[name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		resp := PingResponse{Replies: replies}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write ping response", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

func NewWordsHandler(log *slog.Logger, norm core.Normalizer, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "missing phrase", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		wordsList, err := norm.Norm(ctx, phrase)
		if err != nil {
			handleError(w, err)
			return
		}

		resp := map[string]interface{}{
			"words": wordsList,
			"total": len(wordsList),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("failed to write response", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrUpdateInProgress):
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, core.ErrBadArguments):
		http.Error(w, "phrase too large", http.StatusBadRequest)
	case errors.Is(err, core.ErrServiceUnavailable):
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	default:
		http.Error(w, "internal service error", http.StatusInternalServerError)
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		if err := updater.Update(ctx); err != nil {
			handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

type ServiceStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		stats, err := updater.Stats(ctx)
		if err != nil {
			handleError(w, err)
			return
		}

		statsJson := ServiceStats{
			WordsTotal:    stats.WordsTotal,
			WordsUnique:   stats.WordsUnique,
			ComicsFetched: stats.ComicsFetched,
			ComicsTotal:   stats.ComicsTotal,
		}

		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(statsJson); err != nil {
			log.Error("failed to write response", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		status, err := updater.Status(ctx)
		if err != nil {
			handleError(w, err)
			return
		}

		reply := map[string]string{
			"status": string(status),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(reply); err != nil {
			log.Error("cannot encode status reply")
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HTTPConfig.Timeout)
		defer cancel()

		if err := updater.Drop(ctx); err != nil {
			handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
