package handler

import (
	"bank-api/internal/dto"
	"bank-api/internal/middleware"
	"net/http"

	"github.com/gorilla/mux"
)

// SetupRouter собирает публичные, защищённые и административные эндпоинты приложения
func SetupRouter(handlers *Handlers, authMW mux.MiddlewareFunc, loggingMW mux.MiddlewareFunc, recoveryMW mux.MiddlewareFunc) *mux.Router {
	r := mux.NewRouter()
	r.Use(recoveryMW)
	r.Use(loggingMW)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dto.WriteJSON(w, http.StatusNotFound, dto.APIResponse{
			Success: false,
			Error:   "Эндпоинт " + r.URL.Path + " не существует",
		})
	})
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dto.WriteJSON(w, http.StatusMethodNotAllowed, dto.APIResponse{
			Success: false,
			Error:   "Метод " + r.Method + " не поддерживается для эндпоинта " + r.URL.Path,
		})
	})

	// Публичные и защищённые эндпоинты находятся в одном пространстве /api
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/register", handlers.Auth.Register).Methods("POST")
	api.HandleFunc("/login", handlers.Auth.Login).Methods("POST")

	securedAPI := api.PathPrefix("").Subrouter()
	securedAPI.Use(authMW)

	securedAPI.HandleFunc("/2fa/setup", handlers.Auth.Generate2FA).Methods("POST")
	securedAPI.HandleFunc("/2fa/enable", handlers.Auth.Enable2FA).Methods("POST")
	securedAPI.HandleFunc("/2fa/disable", handlers.Auth.Disable2FA).Methods("POST")
	securedAPI.HandleFunc("/admin/bootstrap", handlers.Auth.BootstrapAdmin).Methods("POST")

	securedAPI.HandleFunc("/accounts", handlers.Account.CreateAccount).Methods("POST")
	securedAPI.HandleFunc("/accounts", handlers.Account.ListAccounts).Methods("GET")
	securedAPI.HandleFunc("/accounts/predict", handlers.Analytics.PredictAllBalances).Methods("GET")
	securedAPI.HandleFunc("/accounts/{id}", handlers.Account.GetAccount).Methods("GET")
	securedAPI.HandleFunc("/accounts/{id}/deposit", handlers.Account.Deposit).Methods("POST")
	securedAPI.HandleFunc("/accounts/{id}/withdraw", handlers.Account.Withdraw).Methods("POST")
	securedAPI.HandleFunc("/accounts/{id}/income-expense", handlers.Analytics.IncomeExpenseByAccount).Methods("GET")
	securedAPI.HandleFunc("/accounts/{id}/credit-load", handlers.Analytics.CreditLoadByAccount).Methods("GET")
	securedAPI.HandleFunc("/accounts/{id}/analytics", handlers.Analytics.AccountSummary).Methods("GET")
	securedAPI.HandleFunc("/accounts/{id}/predict", handlers.Analytics.PredictBalance).Methods("GET")

	securedAPI.HandleFunc("/cards", handlers.Card.IssueCard).Methods("POST")
	securedAPI.HandleFunc("/cards", handlers.Card.ListCards).Methods("GET")
	securedAPI.HandleFunc("/cards/{id}", handlers.Card.GetCard).Methods("GET")
	securedAPI.HandleFunc("/cards/{id}/pay", handlers.Card.Pay).Methods("POST")

	securedAPI.HandleFunc("/transfer", handlers.Transfer.Transfer).Methods("POST")

	securedAPI.HandleFunc("/credits", handlers.Credit.Apply).Methods("POST")
	securedAPI.HandleFunc("/credits", handlers.Credit.List).Methods("GET")
	securedAPI.HandleFunc("/credits/{id}/schedule", handlers.Credit.GetSchedule).Methods("GET")
	securedAPI.HandleFunc("/credits/{id}/pay", handlers.Credit.PayNext).Methods("POST")

	securedAPI.HandleFunc("/analytics", handlers.Analytics.Summary).Methods("GET")
	securedAPI.HandleFunc("/analytics/income-expense", handlers.Analytics.IncomeExpense).Methods("GET")
	securedAPI.HandleFunc("/analytics/credit-load", handlers.Analytics.CreditLoad).Methods("GET")

	// Эндпоинты для администратора
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(authMW, middleware.AdminMiddleware)
	admin.HandleFunc("/users", handlers.Admin.ListUsers).Methods("GET")
	admin.HandleFunc("/users/{id}/block", handlers.Admin.BlockUser).Methods("POST")

	return r
}
