package http

import (
	"net/http"
	"strconv"
	"strings"

	"Connect/internal/domain"
	"Connect/internal/dto"
)

// HandleModels handles both model discovery listing and individual model profile retrieval by ID.
func (h *HTTPHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 1. Role-based backend check: If caller has role 'model', return list of online/available callers!
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if token != "" {
		if u, err := h.authUC.ValidateToken(token); err == nil && u != nil && u.Role == domain.RoleModel {
			users, err := h.authUC.ListAvailableUsers()
			if err != nil {
				SendError(w, http.StatusInternalServerError, err.Error())
				return
			}
			SendJSON(w, http.StatusOK, "Available online callers fetched successfully", users)
			return
		}
	}

	// 2. Check if this is a single model profile request: e.g. /api/models/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/models")
	path = strings.TrimPrefix(path, "/")
	if path != "" && !strings.Contains(path, "/") {
		// Specific subroutes like favorites are handled by dedicated routes, but verify here
		if path != "favorite" && path != "favorites" && path != "favourite" && path != "favourites" && path != "favorite-ids" && path != "onboarding" && path != "me" {
			modelID := path
			model, err := h.authUC.GetModelByID(modelID)
			if err != nil {
				SendError(w, http.StatusNotFound, "Model not found")
				return
			}
			SendJSON(w, http.StatusOK, "Model profile fetched successfully", model)
			return
		}
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

