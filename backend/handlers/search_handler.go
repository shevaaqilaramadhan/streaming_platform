package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"watchparty-backend/services"
)

// HandleSearch processes search queries across IDLIX, Samehadaku, and other providers.
// GET /api/search?q={query}&provider={idlix|samehadaku|all}
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{},
			"total":   0,
		})
		return
	}

	provider := r.URL.Query().Get("provider")
	results := services.SearchAll(q, provider)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   q,
		"results": results,
		"total":   len(results),
	})
}

// HandleCatalogEpisodes fetches episodes list for a series.
// GET /api/catalog/episodes?provider={idlix}&slug={slug}&season={n}
func HandleCatalogEpisodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.URL.Query().Get("provider")
	slug := r.URL.Query().Get("slug")
	seasonStr := r.URL.Query().Get("season")

	season := 1
	if s, err := strconv.Atoi(seasonStr); err == nil && s > 0 {
		season = s
	}

	episodes, err := services.GetSeriesEpisodes(provider, slug, season)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider": provider,
		"slug":     slug,
		"season":   season,
		"episodes": episodes,
		"total":    len(episodes),
	})
}
