package models

type Cart struct {
	CartID string `json:"cart_id"`
	UID    string `json:"uuid,omitempty"`
}

type CartItem struct {
	ItemID   string `json:"iid"`
	CartID   string `json:"cart_id"`
	BID      string `json:"bid,omitempty"`
	Quantity int    `json:"count,omitempty"`
}
type Book struct {
	BID     string `json:"bid,omitempty"`
	Lable   string `json:"lable" validate:"required,min=3"`
	Author  string `json:"author" validate:"required,min=5"`
	Desc    string `json:"desc" validate:"required,min=10"`
	Age     int    `json:"age" validate:"required"`
	Count   int    `json:"count,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
}
