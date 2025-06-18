package models

type Cart struct {
	CartID string `json:"cart_id"`
	UID    string `json:"uuid,omitempty"`
}

type CartItem struct {
	ItemID string `json:"iid,omitempty"`
	CartID string `json:"cart_id"`
	BID    string `json:"bid,omitempty"`
}
type Book struct {
	BID    string `json:"bid,omitempty"`
	Lable  string `json:"lable" validate:"required,min=3"`
	Author string `json:"author" validate:"required,min=5"`
	Desc   string `json:"desc" validate:"required,min=10"`
	Age    int    `json:"age" validate:"required"`
	Genre  string `json:"genre"`
	Rating int    `json:"rating"`
}
