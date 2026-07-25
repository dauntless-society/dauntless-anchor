package handlers

import "net/http"

func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/register", s.Register)
	mux.HandleFunc("/api/v1/auth/challenge", s.Challenge)
	mux.HandleFunc("/api/v1/auth/verify", s.Verify)

	// Protected (ADMIN/FOUNDER)
	mux.Handle("/api/v1/auth/approve", s.Middleware(RequireRoles(http.HandlerFunc(s.Approve), "ADMIN", "FOUNDER")))
	mux.Handle("/api/v1/auth/revoke", s.Middleware(RequireRoles(http.HandlerFunc(s.Revoke), "ADMIN")))
}
