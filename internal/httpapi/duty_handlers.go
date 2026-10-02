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
	writeJSON(w, http.StatusCreated, contracts.OperatorResponse{Operator: operatorView(operator)})
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

func (h *handler) createArea(w http.ResponseWriter, r *http.Request) {
	var request contracts.CreateAreaRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDomainError(w, err)
		return
	}

	area, err := h.service.CreateArea(r.Context(), request)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contracts.AreaResponse{Area: areaView(area)})
}

func (h *handler) area(w http.ResponseWriter, r *http.Request) {
	area, err := h.service.Area(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.AreaResponse{Area: areaView(area)})
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
	writeJSON(w, http.StatusCreated, contracts.ShiftResponse{Shift: shiftView(shift, h.service.Now())})
}

func (h *handler) shift(w http.ResponseWriter, r *http.Request) {
	shift, err := h.service.Shift(r.Context(), r.PathValue("id"))
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

func (h *handler) createAssignment(w http.ResponseWriter, r *http.Request) {
	var request contracts.CreateAssignmentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDomainError(w, err)
		return
	}

	assignment, err := h.service.CreateAssignment(r.Context(), request)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contracts.AssignmentResponse{Assignment: assignmentView(assignment)})
}

func (h *handler) assignment(w http.ResponseWriter, r *http.Request) {
	assignment, err := h.service.Assignment(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.AssignmentResponse{Assignment: assignmentView(assignment)})
}
