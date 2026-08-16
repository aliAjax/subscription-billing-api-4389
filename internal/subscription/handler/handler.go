package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"subscription-billing-api/internal/subscription/model"
	"subscription-billing-api/internal/subscription/service"
	"subscription-billing-api/pkg/response"
)

type SubscriptionHandler struct {
	service service.SubscriptionService
}

func New(subscriptionService service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: subscriptionService}
}

func (h *SubscriptionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /api/v1/subscriptions", h.create)
	mux.HandleFunc("GET /api/v1/subscriptions", h.list)
	mux.HandleFunc("GET /api/v1/subscriptions/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/subscriptions/{id}/renewal-date", h.updateRenewalDate)
	mux.HandleFunc("DELETE /api/v1/subscriptions/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/subscriptions/monthly-total", h.monthlyExpectedSpend)
	mux.HandleFunc("GET /api/v1/subscriptions/upcoming-charges", h.upcomingCharges)
}

func (h *SubscriptionHandler) health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SubscriptionHandler) create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	subscription, err := h.service.Create(req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, subscription)
}

func (h *SubscriptionHandler) list(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := h.service.List("all")
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"subscriptions": subscriptions,
		"count":         len(subscriptions),
	})
}

func (h *SubscriptionHandler) get(w http.ResponseWriter, r *http.Request) {
	subscription, err := h.service.Get(r.PathValue("id"))
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, subscription)
}

func (h *SubscriptionHandler) updateRenewalDate(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateRenewalDateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	subscription, err := h.service.UpdateRenewalDate(r.PathValue("id"), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, subscription)
}

func (h *SubscriptionHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.PathValue("id")); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SubscriptionHandler) monthlyExpectedSpend(w http.ResponseWriter, _ *http.Request) {
	report, err := h.service.MonthlyExpectedSpend()
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, report)
}

func (h *SubscriptionHandler) upcomingCharges(w http.ResponseWriter, r *http.Request) {
	days := 7
	if rawDays := r.URL.Query().Get("days"); rawDays != "" {
		parsedDays, err := strconv.Atoi(rawDays)
		if err != nil || parsedDays < 1 || parsedDays > 365 {
			response.Error(w, http.StatusBadRequest, "validation_error", "days must be an integer between 1 and 365")
			return
		}
		days = parsedDays
	}

	report, err := h.service.UpcomingCharges(days)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, report)
}

func (h *SubscriptionHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		response.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, service.ErrNotFound):
		response.Error(w, http.StatusNotFound, "not_found", err.Error())
	default:
		log.Printf("internal error: %v", err)
		response.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}

	return nil
}
