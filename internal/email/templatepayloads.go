package email

type TemplateWrapper struct {
	Title string
	Year  int
	Data  any
}

type SignupThankYouData struct {
	Name string `json:"name"`
	Year int    `json:"year"`
}

type OrderSuccessData struct {
	Name    string  `json:"name"`
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
	Year    int     `json:"year"`
}

type OrderFailedData struct {
	Name    string `json:"name"`
	OrderID string `json:"order_id"`
}

type OrderCancelledData struct {
	Name    string `json:"name"`
	OrderID string `json:"order_id"`
}

type PaymentSuccessData struct {
	Name          string  `json:"name"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Year          int     `json:"year"`
}

type PaymentFailedData struct {
	Name          string `json:"name"`
	TransactionID string `json:"transaction_id"`
}

type PaymentRefundedData struct {
	Name    string  `json:"name"`
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
	Year    int     `json:"year"`
}
