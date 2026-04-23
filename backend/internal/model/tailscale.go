package model

type TailscaleDevice struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Hostname           string   `json:"hostname"`
	OS                 string   `json:"os"`
	Addresses          []string `json:"addresses"` // парсится из API, не отдаётся напрямую
	LastSeen           string   `json:"lastSeen"`
	ConnectedToControl bool     `json:"connectedToControl"`
	Authorized         bool     `json:"authorized"`
}

type TailscaleDevicesResponse struct {
	Devices []TailscaleDevice `json:"devices"`
}

// DeviceResponse — то, что отдаётся клиенту
type DeviceResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Hostname           string   `json:"hostname"`
	OS                 string   `json:"os"`
	Addresses          []string `json:"addresses,omitempty"`
	LastSeen           string   `json:"lastSeen"`
	ConnectedToControl bool     `json:"connectedToControl"`
	Authorized         bool     `json:"authorized"`
}

func (d TailscaleDevice) ToResponse(showIP bool) DeviceResponse {
	r := DeviceResponse{
		ID:                 d.ID,
		Name:               d.Name,
		Hostname:           d.Hostname,
		OS:                 d.OS,
		LastSeen:           d.LastSeen,
		ConnectedToControl: d.ConnectedToControl,
		Authorized:         d.Authorized,
	}
	if showIP {
		r.Addresses = d.Addresses
	}
	return r
}
