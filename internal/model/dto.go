package model

type AccrualStatus string

const (
	StatusCalculatorRegistered AccrualStatus = "REGISTERED"
	StatusCalculatorInvalid    AccrualStatus = "INVALID"
	StatusCalculatorProcessing AccrualStatus = "PROCESSING"
	StatusCalculatorProcessed  AccrualStatus = "PROCESSED"
	StatusCalculatorNoContent  AccrualStatus = "NO_CONTENT"
	StatusCalculatorFailed     AccrualStatus = "CALCULATOR_FAILED"
	StatusAgentFailed          AccrualStatus = "AGENT_FAILED"
)

type DTOAccrualInfo struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual,omitempty"`
}
