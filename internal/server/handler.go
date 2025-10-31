package server

import (
	"github.com/nhdewitt/http-from-tcp/internal/request"
	"github.com/nhdewitt/http-from-tcp/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)
