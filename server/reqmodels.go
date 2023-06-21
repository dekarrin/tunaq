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
	URI            string `json:"uri"`
	ID             string `json:"id,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	Email          string `json:"email,"`
	Role           string `json:"role,omitempty"`
	Created        string `json:"created,omitempty"`
	Modified       string `json:"modified,omitempty"`
	LastLogoutTime string `json:"last_logout,omitempty"`
	LastLoginTime  string `json:"last_login,omitempty"`
}

type UserUpdateRequest struct {
	ID       UpdateString `json:"id,omitempty"`
	Username UpdateString `json:"username,omitempty"`
	Password UpdateString `json:"password,omitempty"`
	Email    UpdateString `json:"email,"`
	Role     UpdateString `json:"role,omitempty"`
}

type UpdateString struct {
	Update bool   `json:"u,omitempty"`
	Value  string `json:"v,omitempty"`
}
