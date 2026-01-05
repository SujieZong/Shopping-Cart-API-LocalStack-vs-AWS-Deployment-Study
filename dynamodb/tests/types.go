package main

// Shared request/response types for shopping cart API tests

type CreateCartRequest struct {
	CustomerID int `json:"customer_id"`
}

type CreateCartResponse struct {
	ShoppingCartID int64 `json:"shopping_cart_id"`
}

type AddItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}
