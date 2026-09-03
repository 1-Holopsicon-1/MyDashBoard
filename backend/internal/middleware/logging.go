package middleware

import "github.com/go-chi/chi/v5/middleware"

// Logger реэкспортирует chi Logger для удобства.
var Logger = middleware.Logger
