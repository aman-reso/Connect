package http

import (
	"net/http"
	"strconv"

	"Connect/internal/dto"
)

// HandleModels handles model discovery and filtering.
func (h *HTTPHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, _ := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	maxDist, _ := strconv.ParseFloat(r.URL.Query().Get("max_distance_km"), 64)

	filter := &dto.ModelFilterQuery{
		Filter:        r.URL.Query().Get("filter"),
		Lat:           lat,
		Lng:           lng,
		MaxDistanceKM: maxDist,
		City:          r.URL.Query().Get("city"),
		State:         r.URL.Query().Get("state"),
		Gender:        r.URL.Query().Get("gender"),
		Language:      r.URL.Query().Get("language"),
		Interest:      r.URL.Query().Get("interest"),
		SortBy:        r.URL.Query().Get("sort_by"),
		Page:          page,
		Limit:         limit,
	}

	models, err := h.authUC.ListModelsAdvanced(filter)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Models fetched successfully", models)
}
