package usecase_test

import (
	"testing"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/mapper"
	"Connect/internal/repository/memory"
	"Connect/internal/usecase"
)

func TestModelOnboardingFlow(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	onboardUC := usecase.NewModelOnboardingUseCase(store.Onboarding, store.Users, m)

	modelResp, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9777700001", Name: "AspiringModel", Role: domain.RoleModel})

	// 1. Test Underage Rejection (< 18)
	_, err := onboardUC.SubmitOnboarding(modelResp.User, &dto.ModelOnboardingRequest{
		DisplayName:     "YoungStar",
		Age:             17,
		VoiceRatePerMin: 15.0,
		PayoutUPI:       "model@upi",
	})
	if err == nil {
		t.Fatalf("Expected underage model registration (<18) to be rejected")
	}

	// 2. Test Valid Onboarding Submission
	onboardResp, err := onboardUC.SubmitOnboarding(modelResp.User, &dto.ModelOnboardingRequest{
		DisplayName:     "Star Host",
		Bio:             "Professional musician and conversationalist 🎵",
		Age:             22,
		Gender:          "Female",
		Languages:       "English, Hindi",
		Interests:       "Music, Travel, Fitness",
		VoiceRatePerMin: 25.0,
		GroupRatePerMin: 12.0,
		ChatRatePerMsg:  2.0,
		PayoutUPI:       "star@okaxis",
	})
	if err != nil {
		t.Fatalf("Model onboarding failed: %v", err)
	}

	if onboardResp.Profile.Status != domain.OnboardingStatusApproved {
		t.Fatalf("Expected status 'approved', got '%s'", onboardResp.Profile.Status)
	}

	if onboardResp.Profile.VoiceRatePerMin != 25.0 {
		t.Fatalf("Expected rate ₹25.00, got ₹%.2f", onboardResp.Profile.VoiceRatePerMin)
	}

	// 3. Verify User profile updated in storage
	statusResp, err := onboardUC.GetOnboardingStatus(modelResp.User.ID)
	if err != nil || statusResp.Profile.DisplayName != "Star Host" {
		t.Fatalf("Profile persistence failed")
	}
}

func TestModelReportingAndAutomatedSafetyShield(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	reportUC := usecase.NewReportUseCase(store.Reports, store.Onboarding, store.Users, m)

	modelResp, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9777700002", Name: "BadActor", Role: domain.RoleModel})
	u1, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9777700003", Name: "User1", Role: domain.RoleUser})
	u2, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9777700004", Name: "User2", Role: domain.RoleUser})
	u3, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9777700005", Name: "User3", Role: domain.RoleUser})

	// Report 1
	r1, err := reportUC.CreateReport(u1.User, &dto.CreateReportRequest{
		ModelID:     modelResp.User.ID,
		Category:    domain.ReportCategoryHarassment,
		Description: "Used offensive language during call",
	})
	if err != nil || r1.Report.Status != domain.ReportStatusSubmitted {
		t.Fatalf("Report 1 failed: %v", err)
	}

	// Report 2
	_, _ = reportUC.CreateReport(u2.User, &dto.CreateReportRequest{
		ModelID:     modelResp.User.ID,
		Category:    domain.ReportCategoryInappropriate,
		Description: "Inappropriate behavior",
	})

	// Report 3: Should trigger automated safety shield suspension
	r3, err := reportUC.CreateReport(u3.User, &dto.CreateReportRequest{
		ModelID:     modelResp.User.ID,
		Category:    domain.ReportCategoryHarassment,
		Description: "Continuous abuse",
	})
	if err != nil {
		t.Fatalf("Report 3 failed: %v", err)
	}

	// Verify model reports count
	listResp, err := reportUC.GetReportsForModel(modelResp.User.ID)
	if err != nil || listResp.Count != 3 {
		t.Fatalf("Expected 3 reports in history, got %d", listResp.Count)
	}

	// Verify automated suspension
	if r3.Message != "Thank you. The reported profile has been temporarily flagged for immediate review." {
		t.Fatalf("Expected auto-flag suspension notification")
	}
}
