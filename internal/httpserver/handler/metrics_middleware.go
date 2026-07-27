package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ZeroGravity-82/goph-profile/internal/observability"
)

// withMetrics записывает HTTP-метрики после обработки запроса.
func withMetrics(metrics *observability.HTTPRequestMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			metrics.RecordHTTPRequestStarted(r.Context(), r.Method)
			defer metrics.RecordHTTPRequestFinished(r.Context(), r.Method)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			metrics.RecordHTTPRequest(
				r.Context(),
				r.Method,
				httpRoutePattern(r),
				status,
				time.Since(startedAt),
				ww.BytesWritten(),
			)
		})
	}
}

// httpRoutePattern возвращает шаблон chi-маршрута для HTTP-метрик.
//
// Middleware вызывается после обработки запроса, поэтому chi уже успевает записать найденный маршрут в RouteContext.
// В метрики пишется именно шаблон вроде /api/v1/avatars/{avatar_id}, а не фактический URL запроса: так разные
// идентификаторы не создают отдельные временные ряды. Если запрос обработан без chi-маршрута, используется "unknown".
func httpRoutePattern(r *http.Request) string {
	routeContext := chi.RouteContext(r.Context())
	if routeContext == nil {
		return "unknown"
	}
	route := routeContext.RoutePattern()
	if route == "" {
		return "unknown"
	}
	return route
}
