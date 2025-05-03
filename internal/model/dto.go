package model

const (
	StatusCalculatorRegistered string = "REGISTERED"
	StatusCalculatorInvalid    string = "INVALID"
	StatusCalculatorProcessing string = "PROCESSING"
	StatusCalculatorProcessed  string = "PROCESSED"
)

type DTOAccrualInfo struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual,omitempty"`
}
