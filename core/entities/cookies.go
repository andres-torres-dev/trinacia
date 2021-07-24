package entities

// Cookie information of website user
type Cookie struct {
	ID             string `json:"id"`
	IP             string `json:"ip"`
	DeviceID       string `json:"device_id"`
	CreationTime   string `json:"creation_time"`
	LastAccessTime string `json:"last_access"`
	UserID         string `json:"user_id"`
	Persona        string `json:"persona"`
	Company        string `json:"company"`
}
