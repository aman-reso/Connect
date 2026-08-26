package usecase_test

import (
	"testing"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/mapper"
	"Connect/internal/repository/memory"
	"Connect/internal/usecase"
)

func TestAuthAndSignupBonusUseCase(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	walletUC := usecase.NewWalletUseCase(store.Wallets, m)

	resp, err := authUC.RegisterOrLogin(&dto.RegisterRequest{
		Phone: "9999900001",
		Name:  "Test Client",
		Role:  domain.RoleUser,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if !resp.IsNewUser {
		t.Fatalf("Expected new user")
	}

	wResp, err := walletUC.GetWallet(resp.User.ID)
	if err != nil {
		t.Fatalf("Failed to fetch wallet: %v", err)
	}

	if wResp.Wallet.Balance != 50.0 {
		t.Fatalf("Expected ₹50.00 welcome bonus, got ₹%.2f", wResp.Wallet.Balance)
	}
}

func TestCallUseCaseBilling(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	callUC := usecase.NewCallUseCase(store.Calls, store.Users, store.Wallets, m)
	walletUC := usecase.NewWalletUseCase(store.Wallets, m)

	callerResp, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9999900002", Name: "Caller", Role: domain.RoleUser})
	modelsResp, _ := authUC.ListModels()
	targetModel := modelsResp.Models[0]

	callRecord, err := callUC.InitiateCall(callerResp.User, targetModel.ID)
	if err != nil {
		t.Fatalf("Failed to initiate call: %v", err)
	}

	_, err = callUC.AcceptCall(callRecord.ID)
	if err != nil {
		t.Fatalf("Failed to accept call: %v", err)
	}

	cost, dur, err := callUC.EndCall(callRecord.ID, "Normal hangup")
	if err != nil || dur < 0 || cost < 0 {
		t.Fatalf("Failed to end call: %v", err)
	}

	history, _ := callUC.GetHistory(callerResp.User.ID)
	if history.Count != 1 {
		t.Fatalf("Expected 1 history record, got %d", history.Count)
	}

	wResp, _ := walletUC.GetWallet(callerResp.User.ID)
	if wResp.Wallet.Balance > 50.0 {
		t.Fatalf("Balance was not deducted")
	}
}

func TestFavoriteUseCase(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	favUC := usecase.NewFavoriteUseCase(store.Favorites, store.Users, m)

	userResp, err := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9999900003", Name: "Fav User", Role: domain.RoleUser})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	modelsResp, err := authUC.ListModels()
	if err != nil || len(modelsResp.Models) == 0 {
		t.Fatalf("Failed to get models")
	}
	model1 := modelsResp.Models[0]
	model2 := modelsResp.Models[1]

	// 1. Initial favorites count should be 0
	ids, err := favUC.GetFavoriteIDs(userResp.User.ID)
	if err != nil || len(ids) != 0 {
		t.Fatalf("Expected 0 favorites, got %d", len(ids))
	}

	// 2. Mark model1 as favorite
	favResp, err := favUC.ToggleFavorite(userResp.User.ID, model1.ID)
	if err != nil {
		t.Fatalf("Failed to toggle favorite: %v", err)
	}
	if !favResp.IsFavorite {
		t.Fatalf("Expected model1 to be marked as favorite")
	}

	// Verify IsFavorite
	isFav, err := favUC.IsFavorite(userResp.User.ID, model1.ID)
	if err != nil || !isFav {
		t.Fatalf("Expected IsFavorite to be true")
	}

	// 3. Mark model2 as favorite
	_, err = favUC.ToggleFavorite(userResp.User.ID, model2.ID)
	if err != nil {
		t.Fatalf("Failed to toggle favorite model2: %v", err)
	}

	// 4. Get favorites list
	listResp, err := favUC.GetFavorites(userResp.User.ID)
	if err != nil {
		t.Fatalf("Failed to get favorites: %v", err)
	}
	if listResp.Count != 2 {
		t.Fatalf("Expected 2 favorites, got %d", listResp.Count)
	}

	// 5. Unmark model1 (Toggle off)
	unfavResp, err := favUC.ToggleFavorite(userResp.User.ID, model1.ID)
	if err != nil {
		t.Fatalf("Failed to untoggle favorite: %v", err)
	}
	if unfavResp.IsFavorite {
		t.Fatalf("Expected model1 to be removed from favorites")
	}

	// Verify remaining count
	ids, err = favUC.GetFavoriteIDs(userResp.User.ID)
	if err != nil || len(ids) != 1 || ids[0] != model2.ID {
		t.Fatalf("Expected 1 favorite (model2), got %v", ids)
	}

	// 6. Validation: self-favorite should fail
	_, err = favUC.ToggleFavorite(userResp.User.ID, userResp.User.ID)
	if err == nil {
		t.Fatalf("Expected error when favoriting self")
	}

	// 7. Validation: invalid model ID should fail
	_, err = favUC.ToggleFavorite(userResp.User.ID, "non-existent-id")
	if err == nil {
		t.Fatalf("Expected error when favoriting non-existent model")
	}
}

func TestModelDiscoveryFilteringNearbyAndPagination(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)

	// 1. Test Nearby Discovery (Delhi coordinates: 28.6139, 77.2090)
	nearbyResp, err := authUC.ListModelsAdvanced(&dto.ModelFilterQuery{
		Filter: "nearby",
		Lat:    28.6139,
		Lng:    77.2090,
		Page:   1,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListModelsAdvanced nearby failed: %v", err)
	}
	if nearbyResp.Count == 0 {
		t.Fatalf("Expected nearby models, got 0")
	}
	if nearbyResp.Models[0].DistanceKM == nil {
		t.Fatalf("Expected DistanceKM to be computed for nearby search")
	}
	if *nearbyResp.Models[0].DistanceKM > 1.0 {
		t.Fatalf("Expected Delhi model (Aanya) to be closest (~0 km), got %.2f km", *nearbyResp.Models[0].DistanceKM)
	}

	// 2. Test Age Filter (e.g. MinAge 23)
	ageResp, err := authUC.ListModelsAdvanced(&dto.ModelFilterQuery{
		Filter: "all",
		MinAge: 23,
		MaxAge: 30,
		Page:   1,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListModelsAdvanced age filter failed: %v", err)
	}
	for _, mod := range ageResp.Models {
		if mod.Age < 23 || mod.Age > 30 {
			t.Fatalf("Expected model age between 23 and 30, got %d for %s", mod.Age, mod.Name)
		}
	}

	// 3. Test New Models Filter
	newResp, err := authUC.ListModelsAdvanced(&dto.ModelFilterQuery{
		Filter: "new",
		Page:   1,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListModelsAdvanced new filter failed: %v", err)
	}
	if newResp.Count == 0 {
		t.Fatalf("Expected new models list, got 0")
	}

	// 4. Test Top Models Filter
	topResp, err := authUC.ListModelsAdvanced(&dto.ModelFilterQuery{
		Filter: "top",
		Page:   1,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListModelsAdvanced top filter failed: %v", err)
	}
	if topResp.Count == 0 || topResp.Models[0].Rating < 4.8 {
		t.Fatalf("Expected top models with high rating")
	}

	// 5. Test Pagination (Page 1 vs Page 2 with limit 2)
	p1Resp, _ := authUC.ListModelsAdvanced(&dto.ModelFilterQuery{
		Page:  1,
		Limit: 2,
	})
	p2Resp, _ := authUC.ListModelsAdvanced(&dto.ModelFilterQuery{
		Page:  2,
		Limit: 2,
	})
	if len(p1Resp.Models) != 2 || len(p2Resp.Models) != 2 {
		t.Fatalf("Expected 2 models per page, got P1=%d, P2=%d", len(p1Resp.Models), len(p2Resp.Models))
	}
	if p1Resp.Models[0].ID == p2Resp.Models[0].ID {
		t.Fatalf("Page 1 and Page 2 should not have identical models: P1[0]=%s, P2[0]=%s", p1Resp.Models[0].ID, p2Resp.Models[0].ID)
	}
	if !p1Resp.Pagination.HasNext {
		t.Fatalf("Expected Page 1 to have HasNext=true")
	}
}

