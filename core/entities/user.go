package entities

// User is a trinacia user
type User struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	CreationTime string `json:"creation_time"`
}

// Payment struct contains information on validated payment
type Payment struct {
	BillingToken     string `json:"billing_token"`
	FacilitatorToken string `json:"facilitator_token"`
	OrderID          string `json:"order_id"`
	SubscritionID    string `json:"subscription_id"`
}

// Facebook information related to user
type Facebook struct {
	ID          string      `json:"id"`
	Pages       []Page      `json:"pages"`
	AdAccounts  []AdAccount `json:"ad_accounts"`
	AccessToken string      `json:"access_token"`
}

// Page information about a facebook page
type Page struct {
	Category    string      `json:"category"`
	Name        string      `json:"name"`
	ID          string      `json:"id"`
	Instagram   []Instagram `json:"instagram,omitempty"`
	AccessToken string      `json:"access_token"`
}

// Instagram page data
type Instagram struct {
	ID   string `json:"id"`
	Name string `json:"username"`
}

// AdAccount struct is a facebook ad account data
type AdAccount struct {
	AccountID string `json:"account_id"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Currency  string `json:"currency"`
}
