package contracts

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InterlockDecisionView struct {
	Verdict    string `json:"verdict"`
	Verifier   string `json:"verifier"`
	Note       string `json:"note"`
	ReviewedAt string `json:"reviewed_at"`
}

type DutyAttestationView struct {
	ShiftID    string `json:"shift_id"`
	OperatorID string `json:"operator_id"`
	GateID     string `json:"gate_id"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	VerifiedAt string `json:"verified_at"`
}

type GateView struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	Aperture        float64 `json:"aperture"`
	MaxAperture     float64 `json:"max_aperture"`
	ActiveCommandID string  `json:"active_command_id,omitempty"`
	UpdatedAt       string  `json:"updated_at"`
	Version         int64   `json:"version"`
}

type CommandView struct {
	ID               string                 `json:"id"`
	GateID           string                 `json:"gate_id"`
	TargetAperture   float64                `json:"target_aperture"`
	StartingAperture float64                `json:"starting_aperture"`
	Reason           string                 `json:"reason"`
	RequestedBy      string                 `json:"requested_by"`
	Status           string                 `json:"status"`
	Interlock        *InterlockDecisionView `json:"interlock,omitempty"`
	ExecutionToken   string                 `json:"execution_token,omitempty"`
	ExecutedBy       string                 `json:"executed_by,omitempty"`
	Duty             *DutyAttestationView   `json:"duty,omitempty"`
	TerminalReason   string                 `json:"terminal_reason,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	Version          int64                  `json:"version"`
}

type RegisterGateResponse struct {
	Gate GateView `json:"gate"`
}

type GateResponse struct {
	Gate GateView `json:"gate"`
}

type CommandResponse struct {
	Command CommandView `json:"command"`
}

type ReviewCommandResponse struct {
	Command CommandView `json:"command"`
	Gate    GateView    `json:"gate"`
}

type ExecuteCommandResponse struct {
	Command CommandView `json:"command"`
	Gate    GateView    `json:"gate"`
}

type OperatorView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	ValidUntil   string `json:"valid_until,omitempty"`
	RegisteredAt string `json:"registered_at"`
	UpdatedAt    string `json:"updated_at"`
	Version      int64  `json:"version"`
}

type RegisterOperatorResponse struct {
	Operator OperatorView `json:"operator"`
}

type OperatorResponse struct {
	Operator OperatorView `json:"operator"`
}

type ShiftAssignmentView struct {
	OperatorID string `json:"operator_id"`
	AssignedAt string `json:"assigned_at"`
}

type ShiftView struct {
	ID          string                `json:"id"`
	Label       string                `json:"label,omitempty"`
	Status      string                `json:"status"`
	StartsAt    string                `json:"starts_at"`
	EndsAt      string                `json:"ends_at"`
	GateIDs     []string              `json:"gate_ids"`
	Assignments []ShiftAssignmentView `json:"assignments"`
	EndedAt     string                `json:"ended_at,omitempty"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
	Version     int64                 `json:"version"`
}

type CreateShiftResponse struct {
	Shift ShiftView `json:"shift"`
}

type ShiftResponse struct {
	Shift ShiftView `json:"shift"`
}

type GateCoverageView struct {
	GateID         string   `json:"gate_id"`
	Status         string   `json:"status"`
	Covered        bool     `json:"covered"`
	Contended      bool     `json:"contended"`
	ActiveShiftID  string   `json:"active_shift_id,omitempty"`
	OpenShiftIDs   []string `json:"open_shift_ids"`
	MemberShiftIDs []string `json:"member_shift_ids,omitempty"`
	CheckedAt      string   `json:"checked_at"`
}

type GateCoverageResponse struct {
	Coverage GateCoverageView `json:"coverage"`
}
