package handlers

import (
	"encoding/json"
	"net/http"
)

// NotImplemented es un handler temporal para rutas que todavía no están implementadas.
// Mejor que un 404 — deja claro que la ruta existe pero está en construcción.
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "not_implemented",
		"message": "este endpoint está en construcción",
	})
}
