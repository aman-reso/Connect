package http

import (
	"encoding/json"
	"net/http"

	"Connect/internal/domain"
	"Connect/internal/dto"
)

// HandleModelOnboarding applies or gets onboarding status.
func (h *HTTPHandler) HandleModelOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "user_default"
		}
		var req dto.ModelOnboardingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		modelUser := &domain.User{
			ID:   userID,
			Name: req.DisplayName,
		}
		resp, err := h.onboardUC.SubmitOnboarding(modelUser, &req)
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error())
			return
		}
		SendJSON(w, http.StatusOK, "Model onboarding submitted successfully", resp)
		return
	}

	h.HandleGetModelOnboardingStatus(w, r)
}

// HandleGetModelOnboardingStatus returns the user's model status.
func (h *HTTPHandler) HandleGetModelOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		SendError(w, http.StatusBadRequest, "Missing user_id parameter")
		return
	}
	profile, err := h.onboardUC.GetOnboardingStatus(userID)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Application status retrieved", profile)
}

// HandleCreateReport submits an abuse report.
func (h *HTTPHandler) HandleCreateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req dto.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	reporterID := r.URL.Query().Get("user_id")
	if reporterID == "" {
		reporterID = "reporter_default"
	}
	reporter := &domain.User{
		ID:   reporterID,
		Name: "Reporter",
	}
	report, err := h.reportUC.CreateReport(reporter, &req)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendJSON(w, http.StatusCreated, "Report created", report)
}

// HandleGetModelReports lists reports filed against a model.
func (h *HTTPHandler) HandleGetModelReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	modelID := r.URL.Query().Get("model_id")
	if modelID == "" {
		SendError(w, http.StatusBadRequest, "Missing model_id parameter")
		return
	}
	reports, err := h.reportUC.GetReportsForModel(modelID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Model reports retrieved", reports)
}
