package models

type Book struct {
	BID      string `json:"bid,omitempty"`
	Lable    string `json:"lable" validate:"required,min=3"`
	Author   string `json:"author" validate:"required,min=5"`
	Desc     string `json:"desc" validate:"required,min=10"`
	Age      int    `json:"age" validate:"required"`
	Genre    string `json:"genre"`
	Rating   int    `json:"rating"`
	CoverURL string `json:"cover_url,omitempty"`
	PDFURL   string `json:"pdf_url,omitempty"`
}
