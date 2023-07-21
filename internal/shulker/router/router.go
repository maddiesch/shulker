package router

import "net/http"

type Handler map[string]http.Handler

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m, ok := h[r.Method]; !ok {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	} else {
		m.ServeHTTP(w, r)
	}
}
