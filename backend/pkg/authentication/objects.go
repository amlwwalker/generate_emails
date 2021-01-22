package authentication

type GResponse struct {
	State string `form:"state"`
	Code  string `form:"code"`
}

type GError struct {
	Error struct{
		Code int `json:"code"`
		Message string `json:"message"`
		Status string `json:"status"`
		Details string `json:"details"`
		Description string `json:"description"`
	}
}
