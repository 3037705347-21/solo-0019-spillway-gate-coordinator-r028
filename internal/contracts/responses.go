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

type ExecutionDutyView struct {
	ShiftID      string `json:"shift_id"`
	ShiftName    string `json:"shift_name"`
	AssignmentID string `json:"assignment_id"`
	AreaID       string `json:"area_id"`
	OperatorID   string `json:"operator_id"`
	OperatorName string `json:"operator_name"`
}

type GateView struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	AreaID          string  `json:"area_id,omitempty"`
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
	ExecutionDuty    *ExecutionDutyView     `json:"execution_duty,omitempty"`
	TerminalReason   string                 `json:"terminal_reason,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	Version          int64                  `json:"version"`
}

type OperatorView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Qualified bool   `json:"qualified"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Version   int64  `json:"version"`
}

type AreaView struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	GateIDs   []string `json:"gate_ids"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	Version   int64    `json:"version"`
}

type ShiftView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Phase      string `json:"phase"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	EndedEarly bool   `json:"ended_early"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Version    int64  `json:"version"`
}

type AssignmentView struct {
	ID         string `json:"id"`
	ShiftID    string `json:"shift_id"`
	AreaID     string `json:"area_id"`
	OperatorID string `json:"operator_id"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Version    int64  `json:"version"`
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

type OperatorResponse struct {
	Operator OperatorView `json:"operator"`
}

type AreaResponse struct {
	Area AreaView `json:"area"`
}

type ShiftResponse struct {
	Shift ShiftView `json:"shift"`
}

type AssignmentResponse struct {
	Assignment AssignmentView `json:"assignment"`
}
