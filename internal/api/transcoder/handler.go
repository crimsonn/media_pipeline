package transcoder

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service}
}
