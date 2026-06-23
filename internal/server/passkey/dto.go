package passkey

import "time"

type CredentialResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type RenameRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}
