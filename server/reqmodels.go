package server

// note that these are *not* the DAO models; those are distinct and closer to
// the DB format they are in. Rather these are the models that are received from
// and sent to the client.

type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

type UserModel struct {
	URI      string `json:"uri"`
	ID       string `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}
