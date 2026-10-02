package contracts

type RegisterGateRequest struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	MaxAperture     float64 `json:"max_aperture"`
	InitialAperture float64 `json:"initial_aperture"`
}

type RequestCommandRequest struct {
	GateID         string  `json:"gate_id"`
	TargetAperture float64 `json:"target_aperture"`
	Reason         string  `json:"reason"`
	RequestedBy    string  `json:"requested_by"`
}

type ReviewInterlockRequest struct {
	Verdict  string `json:"verdict"`
	Verifier string `json:"verifier"`
	Note     string `json:"note"`
}

type ExecuteCommandRequest struct {
	OperatorID    string `json:"operator_id"`
	Token         string `json:"token"`
	SimulateFault bool   `json:"simulate_fault"`
}

type RegisterOperatorRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateAreaRequest struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	GateIDs []string `json:"gate_ids"`
}

type CreateShiftRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

type CreateAssignmentRequest struct {
	ShiftID    string `json:"shift_id"`
	AreaID     string `json:"area_id"`
	OperatorID string `json:"operator_id"`
}
