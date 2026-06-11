package handlers

// Tipos usados apenas para documentar no Swagger as respostas que, em código,
// são montadas com gin.H (que o swag não consegue introspectar).

type meResponse struct {
	User userResponse `json:"user"`
}

type categoryListResponse struct {
	Categories []categoryResponse `json:"categories"`
}

type ticketListResponse struct {
	Tickets []ticketResponse `json:"tickets"`
}

type commentListResponse struct {
	Comments []commentResponse `json:"comments"`
}

type teamListResponse struct {
	Teams []teamResponse `json:"teams"`
}

type ticketEventListResponse struct {
	Events []ticketEventResponse `json:"events"`
}

type slaPolicyListResponse struct {
	Policies []slaPolicyResponse `json:"policies"`
}

type notificationListResponse struct {
	Notifications []notificationResponse `json:"notifications"`
}

type unreadCountResponse struct {
	Count int64 `json:"count"`
}
