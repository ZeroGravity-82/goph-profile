package handler

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// withHTTPRouteTracing добавляет в HTTP span шаблон маршрута, найденный chi.
func withHTTPRouteTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer recordHTTPRouteSpan(r)

		next.ServeHTTP(w, r)
	})
}

// recordHTTPRouteSpan обновляет имя HTTP span и записывает атрибут http.route после выбора маршрута chi.
//
// Middleware выполняется внутри otelhttp.NewHandler: otelhttp уже создал span, а chi после обработки запроса уже
// записал найденный шаблон маршрута в RouteContext.
func recordHTTPRouteSpan(r *http.Request) {
	route := httpRoutePattern(r)
	if route == unknownHTTPRoute {
		return
	}

	span := trace.SpanFromContext(r.Context())
	if !span.IsRecording() {
		return
	}

	span.SetName(r.Method + " " + route)
	span.SetAttributes(attribute.String("http.route", route))
}
