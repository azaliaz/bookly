package models

type User struct {
	UID    string `json:"uuid,omitempty"`
	CartID string `json:"cart_id"`
	Email  string `json:"email" validate:"required,email"`
	Pass   string `json:"pass" validate:"required,min=8"`
	Age    int    `json:"age" validate:"required,gte=16"`
	Role   string `json:"role" validate:"required"`
}
