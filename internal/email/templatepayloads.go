package email

type SignupThankYouData struct {
	Name string
	Year int
}

type OrderSuccessData struct {
	Name    string
	OrderID string
	Amount  float64
	Year    int
}

type OrderFailedData struct {
	Name    string
	OrderID string
}

type OrderCancelled struct {
  Name string
  OrderID string
}

type PaymentSuccessData struct {
	Name          string
	TransactionID string
	Amount        float64
	Year          int
}

type PaymentFailedData struct {
  Name string
  TransactionID string
}

type PaymentRefundedData struct {
	Name    string
	OrderID string
	Amount  float64
	Year    int
}
