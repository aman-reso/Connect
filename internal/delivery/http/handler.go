package http

import "Connect/internal/usecase"

// HTTPHandler aggregates usecases and serves HTTP endpoints.
type HTTPHandler struct {
	authUC     *usecase.AuthUseCase
	walletUC   *usecase.WalletUseCase
	callUC     *usecase.CallUseCase
	roomUC     *usecase.RoomUseCase
	paymentUC  *usecase.PaymentUseCase
	onboardUC  *usecase.ModelOnboardingUseCase
	reportUC   *usecase.ReportUseCase
	favoriteUC *usecase.FavoriteUseCase
}

// NewHTTPHandler constructs the unified HTTP delivery controller.
func NewHTTPHandler(
	auth *usecase.AuthUseCase,
	wallet *usecase.WalletUseCase,
	call *usecase.CallUseCase,
	room *usecase.RoomUseCase,
	payment *usecase.PaymentUseCase,
	onboard *usecase.ModelOnboardingUseCase,
	report *usecase.ReportUseCase,
	favorite *usecase.FavoriteUseCase,
) *HTTPHandler {
	return &HTTPHandler{
		authUC:     auth,
		walletUC:   wallet,
		callUC:     call,
		roomUC:     room,
		paymentUC:  payment,
		onboardUC:  onboard,
		reportUC:   report,
		favoriteUC: favorite,
	}
}
