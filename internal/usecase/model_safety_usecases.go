package usecase

import (
	"errors"
	"fmt"
	"time"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/mapper"
	"Connect/internal/repository"

	"github.com/google/uuid"
)

// ModelOnboardingUseCase 1. Model Onboarding UseCase
type ModelOnboardingUseCase struct {
	onboardRepo repository.ModelOnboardingRepository
	userRepo    repository.UserRepository
	mapper      *mapper.Mapper
}

func NewModelOnboardingUseCase(oRepo repository.ModelOnboardingRepository, uRepo repository.UserRepository, m *mapper.Mapper) *ModelOnboardingUseCase {
	return &ModelOnboardingUseCase{
		onboardRepo: oRepo,
		userRepo:    uRepo,
		mapper:      m,
	}
}

func (uc *ModelOnboardingUseCase) SubmitOnboarding(modelUser *domain.User, req *dto.ModelOnboardingRequest) (*dto.ModelOnboardingResponse, error) {
	if req.Age < 18 {
		return nil, errors.New("you must be at least 18 years old to onboard as a model")
	}
	if req.DisplayName == "" {
		req.DisplayName = modelUser.Name
	}
	if req.VoiceRatePerMin < 1.0 || req.VoiceRatePerMin > 500.0 {
		return nil, errors.New("voice call rate must be between ₹5.00 and ₹500.00 per minute")
	}
	if req.GroupRatePerMin <= 0 {
		req.GroupRatePerMin = req.VoiceRatePerMin * 0.5
	}
	if req.ChatRatePerMsg <= 0 {
		req.ChatRatePerMsg = 1.0
	}
	if req.PayoutUPI == "" && req.PayoutBankAcc == "" {
		return nil, errors.New("either a UPI ID or Bank Account number is required for payout settlements")
	}

	profileID := "prof_" + uuid.New().String()[:10]
	profile := &domain.ModelProfile{
		ID:              profileID,
		UserID:          modelUser.ID,
		DisplayName:     req.DisplayName,
		Bio:             req.Bio,
		AvatarURL:       req.AvatarURL,
		Age:             req.Age,
		Gender:          req.Gender,
		Languages:       req.Languages,
		Interests:       req.Interests,
		VoiceRatePerMin: req.VoiceRatePerMin,
		GroupRatePerMin: req.GroupRatePerMin,
		ChatRatePerMsg:  req.ChatRatePerMsg,
		PayoutUPI:       req.PayoutUPI,
		PayoutBankAcc:   req.PayoutBankAcc,
		PayoutIFSC:      req.PayoutIFSC,
		AudioIntroURL:   req.AudioIntroURL,
		Status:          domain.OnboardingStatusApproved, // Auto-verified upon valid 18+ & payout info
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := uc.onboardRepo.SaveProfile(profile); err != nil {
		return nil, fmt.Errorf("failed to save onboarding profile: %w", err)
	}

	// Update core user rates & bio
	modelUser.Name = profile.DisplayName
	modelUser.Bio = profile.Bio
	if profile.AvatarURL != "" {
		modelUser.AvatarURL = profile.AvatarURL
	}
	modelUser.VoiceRatePerMin = profile.VoiceRatePerMin
	modelUser.GroupRatePerMin = profile.GroupRatePerMin
	modelUser.ChatRatePerMsg = profile.ChatRatePerMsg

	return uc.mapper.ToModelOnboardingResponse(profile, "Model onboarding completed successfully! Your profile is active."), nil
}

func (uc *ModelOnboardingUseCase) GetOnboardingStatus(userID string) (*dto.ModelOnboardingResponse, error) {
	profile, err := uc.onboardRepo.GetProfileByUserID(userID)
	if err != nil {
		return nil, err
	}
	return uc.mapper.ToModelOnboardingResponse(profile, "Profile retrieved"), nil
}

// ReportUseCase 2. User Safety & Model Reporting UseCase
type ReportUseCase struct {
	reportRepo  repository.ReportRepository
	onboardRepo repository.ModelOnboardingRepository
	userRepo    repository.UserRepository
	mapper      *mapper.Mapper
}

func NewReportUseCase(rRepo repository.ReportRepository, oRepo repository.ModelOnboardingRepository, uRepo repository.UserRepository, m *mapper.Mapper) *ReportUseCase {
	return &ReportUseCase{
		reportRepo:  rRepo,
		onboardRepo: oRepo,
		userRepo:    uRepo,
		mapper:      m,
	}
}

func (uc *ReportUseCase) CreateReport(reporter *domain.User, req *dto.CreateReportRequest) (*dto.ReportResponse, error) {
	if req.ModelID == "" {
		return nil, errors.New("model_id is required")
	}
	if req.Category == "" {
		req.Category = domain.ReportCategoryInappropriate
	}
	if req.Description == "" {
		return nil, errors.New("please provide a brief description of the issue")
	}

	modelUser, err := uc.userRepo.GetByID(req.ModelID)
	if err != nil {
		return nil, errors.New("reported model not found")
	}

	reportID := "rep_" + uuid.New().String()[:10]
	report := &domain.ModelReport{
		ID:           reportID,
		ReporterID:   reporter.ID,
		ReporterName: reporter.Name,
		ModelID:      modelUser.ID,
		ModelName:    modelUser.Name,
		CallID:       req.CallID,
		RoomID:       req.RoomID,
		Category:     req.Category,
		Description:  req.Description,
		Status:       domain.ReportStatusSubmitted,
		CreatedAt:    time.Now(),
	}

	if err := uc.reportRepo.CreateReport(report); err != nil {
		return nil, fmt.Errorf("failed to submit report: %w", err)
	}

	// Automated Safety Shield: Increment report count and auto-suspend if reports >= 3
	newReportCount, _ := uc.onboardRepo.IncrementReportCount(modelUser.ID)
	msg := "Thank you. Your report has been submitted to trust & safety moderation."

	if newReportCount >= 3 {
		_ = uc.onboardRepo.SetSuspension(modelUser.ID, true)
		_ = uc.userRepo.SetPresence(modelUser.ID, false, false) // Mark offline
		msg = "Thank you. The reported profile has been temporarily flagged for immediate review."
	}

	return uc.mapper.ToReportResponse(report, msg), nil
}

func (uc *ReportUseCase) GetReportsForModel(modelID string) (*dto.ListReportsResponse, error) {
	reports, err := uc.reportRepo.GetReportsForModel(modelID)
	if err != nil {
		return nil, err
	}
	return uc.mapper.ToListReportsResponse(reports), nil
}

func (uc *ReportUseCase) ResolveReport(req *dto.ResolveReportRequest) (*dto.ReportResponse, error) {
	report, err := uc.reportRepo.GetReportByID(req.ReportID)
	if err != nil {
		return nil, err
	}

	report.Status = domain.ReportStatusResolved
	report.AdminAction = req.AdminAction
	report.AdminNote = req.AdminNote
	now := time.Now()
	report.ResolvedAt = &now

	if err := uc.reportRepo.UpdateReport(report); err != nil {
		return nil, err
	}

	if req.AdminAction == "ban" || req.AdminAction == "suspension" {
		_ = uc.onboardRepo.SetSuspension(report.ModelID, true)
		_ = uc.userRepo.SetPresence(report.ModelID, false, false)
	}

	return uc.mapper.ToReportResponse(report, "Report resolved"), nil
}
