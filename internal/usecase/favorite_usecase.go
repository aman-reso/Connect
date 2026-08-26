package usecase

import (
	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/mapper"
	"Connect/internal/repository"
	"errors"
	"fmt"
)

type FavoriteUseCase struct {
	favRepo  repository.FavoriteRepository
	userRepo repository.UserRepository
	mapper   *mapper.Mapper
}

func NewFavoriteUseCase(favRepo repository.FavoriteRepository, userRepo repository.UserRepository, m *mapper.Mapper) *FavoriteUseCase {
	return &FavoriteUseCase{
		favRepo:  favRepo,
		userRepo: userRepo,
		mapper:   m,
	}
}

func (uc *FavoriteUseCase) ToggleFavorite(userID, modelID string) (*dto.ToggleFavoriteResponse, error) {
	if modelID == "" {
		return nil, errors.New("model_id is required")
	}
	if userID == modelID {
		return nil, errors.New("cannot favorite yourself")
	}

	// Verify model exists and has model role
	targetModel, err := uc.userRepo.GetByID(modelID)
	if err != nil || targetModel == nil {
		return nil, fmt.Errorf("model not found")
	}
	if targetModel.Role != domain.RoleModel {
		return nil, fmt.Errorf("user is not a model")
	}

	isFav, err := uc.favRepo.ToggleFavorite(userID, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle favorite: %w", err)
	}

	msg := "Model added to favorites"
	if !isFav {
		msg = "Model removed from favorites"
	}

	return &dto.ToggleFavoriteResponse{
		ModelID:    modelID,
		IsFavorite: isFav,
		Message:    msg,
	}, nil
}

func (uc *FavoriteUseCase) GetFavorites(userID string) (*dto.FavoriteModelsResponse, error) {
	models, err := uc.favRepo.GetFavoriteModels(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorite models: %w", err)
	}
	return uc.mapper.ToFavoriteModelsResponse(models), nil
}

func (uc *FavoriteUseCase) GetFavoriteIDs(userID string) ([]string, error) {
	ids, err := uc.favRepo.GetFavoriteModelIDs(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorite IDs: %w", err)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

func (uc *FavoriteUseCase) IsFavorite(userID, modelID string) (bool, error) {
	return uc.favRepo.IsFavorite(userID, modelID)
}
