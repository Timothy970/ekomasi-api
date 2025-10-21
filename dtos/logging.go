package dtos

type UserLog struct {
	LogID       string  `json:"log_id"`
	User        *Users  `json:"user"`
	Action      string  `json:"action"`
	Description string  `json:"description,omitempty"`
	Level       string  `json:"level,omitempty"`
	Status      string  `json:"status,omitempty"`
	IPAddress   string  `json:"ip_address,omitempty"`
	UserAgent   *string `json:"user_agent,omitempty"`
	CreatedAt   string  `json:"created_at"`
	Module      string  `json:"module,omitempty"`
}
