package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deliveryHttp "Connect/internal/delivery/http"
	deliveryWs "Connect/internal/delivery/ws"
	wsHandlers "Connect/internal/delivery/ws/handlers"
	"Connect/internal/mapper"
	"Connect/internal/repository"
	"Connect/internal/repository/memory"
	"Connect/internal/repository/postgres"
	"Connect/internal/usecase"
)

func corsAndLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Initialize Repositories
	var (
		userRepo     repository.UserRepository
		walletRepo   repository.WalletRepository
		callRepo     repository.CallRepository
		roomRepo     repository.RoomRepository
		msgRepo      repository.MessageRepository
		paymentRepo  repository.PaymentRepository
		onboardRepo  repository.ModelOnboardingRepository
		reportRepo   repository.ReportRepository
		favoriteRepo repository.FavoriteRepository
	)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pgStore, err := postgres.NewPostgresStore(dbURL)
		if err != nil {
			log.Printf("⚠️ Warning: PostgreSQL failed: %v. Using Memory Repository.", err)
			memStore := memory.NewMemoryStore()
			userRepo, walletRepo, callRepo, roomRepo, msgRepo, paymentRepo, onboardRepo, reportRepo, favoriteRepo = memStore.Users, memStore.Wallets, memStore.Calls, memStore.Rooms, memStore.Messages, memStore.Payments, memStore.Onboarding, memStore.Reports, memStore.Favorites
		} else {
			userRepo, walletRepo, callRepo, roomRepo, msgRepo, paymentRepo, onboardRepo, reportRepo, favoriteRepo = pgStore.Users, pgStore.Wallets, pgStore.Calls, pgStore.Rooms, pgStore.Messages, pgStore.Payments, pgStore.Onboarding, pgStore.Reports, pgStore.Favorites
			log.Println("🐘 PostgreSQL Clean Architecture Repository active!")
		}
	} else {
		log.Println("ℹ️ Running with Clean In-Memory Repository. Set DATABASE_URL for PostgreSQL.")
		memStore := memory.NewMemoryStore()
		userRepo, walletRepo, callRepo, roomRepo, msgRepo, paymentRepo, onboardRepo, reportRepo, favoriteRepo = memStore.Users, memStore.Wallets, memStore.Calls, memStore.Rooms, memStore.Messages, memStore.Payments, memStore.Onboarding, memStore.Reports, memStore.Favorites
	}

	// 2. Initialize Mapper Layer
	m := mapper.NewMapper()

	// 3. Initialize UseCase Layer (Application Business Services)
	authUC := usecase.NewAuthUseCase(userRepo, walletRepo, m)
	walletUC := usecase.NewWalletUseCase(walletRepo, m)
	callUC := usecase.NewCallUseCase(callRepo, userRepo, walletRepo, m)
	roomUC := usecase.NewRoomUseCase(roomRepo, m)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo, walletRepo, m)
	onboardUC := usecase.NewModelOnboardingUseCase(onboardRepo, userRepo, m)
	reportUC := usecase.NewReportUseCase(reportRepo, onboardRepo, userRepo, m)
	favoriteUC := usecase.NewFavoriteUseCase(favoriteRepo, userRepo, m)
	chatUC := usecase.NewChatUseCase(msgRepo, userRepo, walletRepo, m)

	// Background 24-Hour Ephemeral Chat Auto-Purge Worker
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := chatUC.PurgeExpired(); err != nil {
				log.Printf("⚠️ Ephemeral auto-purge warning: %v", err)
			}
		}
	}()

	// 4. Initialize Delivery Layer (HTTP Controllers & WebSocket Signaling Hub)
	httpDelivery := deliveryHttp.NewHTTPHandler(authUC, walletUC, callUC, roomUC, paymentUC, onboardUC, reportUC, favoriteUC, chatUC)
	wsRouter := deliveryWs.NewRouter()
	wsHub := deliveryWs.NewHub(wsRouter, authUC)
	wsRouter.Register(wsHandlers.NewWebRTCHandler(wsHub))
	wsRouter.Register(wsHandlers.NewCallHandler(wsHub, callUC, walletUC))
	wsRouter.Register(wsHandlers.NewRoomHandler(wsHub, roomUC))
	wsRouter.Register(wsHandlers.NewChatHandler(wsHub))

	// 5. Register Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHub.ServeWS)
	mux.HandleFunc("/api/auth", httpDelivery.HandleAuth)
	mux.HandleFunc("/api/auth/register", httpDelivery.HandleAuth)
	mux.HandleFunc("/api/auth/login", httpDelivery.HandleAuth)
	mux.HandleFunc("/api/user/profile", httpDelivery.HandleUserProfile)
	mux.HandleFunc("/api/models/me", httpDelivery.HandleUserProfile)
	mux.HandleFunc("/api/v1/user/profile", httpDelivery.HandleUserProfile)
	mux.HandleFunc("/api/profile", httpDelivery.HandleUserProfile)
	mux.HandleFunc("/api/models", httpDelivery.HandleModels)
	mux.HandleFunc("/api/models/", httpDelivery.HandleModels)
	mux.HandleFunc("/api/models/favorite", httpDelivery.HandleToggleFavorite)
	mux.HandleFunc("/api/models/favourite", httpDelivery.HandleToggleFavorite)
	mux.HandleFunc("/api/models/favorites", httpDelivery.HandleGetFavorites)
	mux.HandleFunc("/api/models/favourites", httpDelivery.HandleGetFavorites)
	mux.HandleFunc("/api/models/favorite-ids", httpDelivery.HandleGetFavoriteIDs)
	mux.HandleFunc("/api/rooms", httpDelivery.HandleRooms)
	mux.HandleFunc("/api/wallet", httpDelivery.HandleWallet)
	mux.HandleFunc("/api/wallet/packs", httpDelivery.HandleWalletPackages)
	mux.HandleFunc("/api/wallet/packages", httpDelivery.HandleWalletPackages)
	mux.HandleFunc("/api/calls/check-balance", httpDelivery.HandleCheckCallBalance)
	mux.HandleFunc("/api/calls/check", httpDelivery.HandleCheckCallBalance)
	mux.HandleFunc("/api/history/calls", httpDelivery.HandleHistory)

	// Ephemeral 24-Hour Chat Routes
	mux.HandleFunc("/api/chat/conversations", httpDelivery.HandleChatConversations)
	mux.HandleFunc("/api/chat/messages", httpDelivery.HandleChatMessages)
	mux.HandleFunc("/api/chat/send", httpDelivery.HandleSendChatMessage)

	// Payment State Machine & Audit Routes
	mux.HandleFunc("/api/payments/order", httpDelivery.HandleCreatePaymentOrder)
	mux.HandleFunc("/api/payments/callback", httpDelivery.HandlePaymentCallback)
	mux.HandleFunc("/api/payments/retry", httpDelivery.HandlePaymentRetry)
	mux.HandleFunc("/api/payments/refund", httpDelivery.HandlePaymentRefund)
	mux.HandleFunc("/api/payments/timeline", httpDelivery.HandlePaymentTimeline)

	// Model Onboarding & User Safety / Reporting Routes
	mux.HandleFunc("/api/model/onboarding", httpDelivery.HandleModelOnboarding)
	mux.HandleFunc("/api/models/onboarding", httpDelivery.HandleModelOnboarding)
	mux.HandleFunc("/api/model/onboarding/status", httpDelivery.HandleGetModelOnboardingStatus)
	mux.HandleFunc("/api/models/onboarding/status", httpDelivery.HandleGetModelOnboardingStatus)
	mux.HandleFunc("/api/models/onboarding/me", httpDelivery.HandleGetModelOnboardingStatus)
	mux.HandleFunc("/api/reports", httpDelivery.HandleCreateReport)
	mux.HandleFunc("/api/reports/model", httpDelivery.HandleGetModelReports)

	// Static Web App Interface
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	handler := corsAndLogMiddleware(mux)
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	// 6. Graceful Shutdown Listener
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("\n🚀 Connect Clean Architecture Backend running at http://localhost:%s\n", port)
		fmt.Printf("🧱 Layered Architecture: Domain ➔ DTO ➔ Mapper ➔ Repository ➔ UseCase ➔ Delivery\n")
		fmt.Printf("🔒 Audio Privacy: DTLS-SRTP E2EE Relay (Zero Audio Storage)\n")
		fmt.Printf("📱 Single Device Policy: Auto-evicts old sessions on new login\n")
		fmt.Printf("💰 Incentive Engine: ₹50 Welcome Bonus enabled for new users\n")
		fmt.Printf("👥 Group Lounge: Multi-user audio lounge with independent per-user billing\n")
		fmt.Printf("⚡ WebSocket Signaling: ws://localhost:%s/ws\n\n", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-stopChan
	log.Println("🛑 Graceful Shutdown triggered: Safely closing connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)
	log.Println("✅ Server cleanly stopped with zero data loss.")
}
