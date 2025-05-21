package dto

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

type AccrualInfo struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}
