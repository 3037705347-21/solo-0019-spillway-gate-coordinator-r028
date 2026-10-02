package httpapi

import (
	"net/http"

	"example.com/spillway-gate-coordinator/internal/contracts"
)

func (h *handler) registerOperator(w http.ResponseWriter, r *http.Request) {
	var request contracts.RegisterOperatorRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDomainError(w, err)
		return
	}

	operator, err := h.service.RegisterOperator(r.Context(), request)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contracts.RegisterOperatorResponse{Operator: operatorView(operator)})
}

func (h *handler) operator(w http.ResponseWriter, r *http.Request) {
	operator, err := h.service.Operator(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.OperatorResponse{Operator: operatorView(operator)})
}

func (h *handler) revokeOperator(w http.ResponseWriter, r *http.Request) {
	operator, err := h.service.RevokeOperator(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.OperatorResponse{Operator: operatorView(operator)})
}

func (h *handler) createShift(w http.ResponseWriter, r *http.Request) {
	var request contracts.CreateShiftRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDomainError(w, err)
		return
	}

	shift, err := h.service.CreateShift(r.Context(), request)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contracts.CreateShiftResponse{Shift: shiftView(shift, h.service.Now())})
}

func (h *handler) shift(w http.ResponseWriter, r *http.Request) {
	shift, err := h.service.Shift(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.ShiftResponse{Shift: shiftView(shift, h.service.Now())})
}

func (h *handler) assignShiftOperator(w http.ResponseWriter, r *http.Request) {
	var request contracts.AssignShiftRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDomainError(w, err)
		return
	}

	shift, err := h.service.AssignShiftOperator(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.ShiftResponse{Shift: shiftView(shift, h.service.Now())})
}

func (h *handler) endShift(w http.ResponseWriter, r *http.Request) {
	shift, err := h.service.EndShift(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.ShiftResponse{Shift: shiftView(shift, h.service.Now())})
}

func (h *handler) gateCoverage(w http.ResponseWriter, r *http.Request) {
	coverage, err := h.service.GateCoverage(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.GateCoverageResponse{Coverage: coverageView(coverage, h.service.Now())})
}
