{{/* Имя chart. */}}
{{- define "goph-profile.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Полное имя ресурсов релиза. */}}
{{- define "goph-profile.fullname" -}}
{{- if contains .Chart.Name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/* Имя chart и его версия для метки helm.sh/chart. */}}
{{- define "goph-profile.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Общие метки ресурсов. */}}
{{- define "goph-profile.labels" -}}
helm.sh/chart: {{ include "goph-profile.chart" . }}
{{ include "goph-profile.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* Общие селекторные метки приложения. */}}
{{- define "goph-profile.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goph-profile.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* Полное имя образа приложения. */}}
{{- define "goph-profile.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{/* Имя общей ConfigMap. */}}
{{- define "goph-profile.configMapName" -}}
{{- printf "%s-config" (include "goph-profile.fullname" .) }}
{{- end }}

{{/* Имя Secret с учётными данными внешних зависимостей. */}}
{{- define "goph-profile.secretName" -}}
{{- printf "%s-secret" (include "goph-profile.fullname" .) }}
{{- end }}

{{/* Имя Secret с TLS-сертификатом Ingress. */}}
{{- define "goph-profile.ingressTLSSecretName" -}}
{{- printf "%s-ingress-tls" (include "goph-profile.fullname" .) }}
{{- end }}

{{/* Имя Traefik Middleware с ограничением частоты запросов. */}}
{{- define "goph-profile.rateLimitMiddlewareName" -}}
{{- printf "%s-server-rate-limit" (include "goph-profile.fullname" .) }}
{{- end }}

{{/* Ссылка на Traefik Middleware в формате провайдера Kubernetes CRD. */}}
{{- define "goph-profile.rateLimitMiddlewareReference" -}}
{{- printf "%s-%s@kubernetescrd" .Release.Namespace (include "goph-profile.rateLimitMiddlewareName" .) }}
{{- end }}

{{/* Имя ресурсов OpenTelemetry Collector. */}}
{{- define "goph-profile.otelCollectorName" -}}
{{- printf "%s-otel-collector" (include "goph-profile.fullname" .) }}
{{- end }}

{{/* Имя ресурсов мигратора. */}}
{{- define "goph-profile.migrateName" -}}
{{- printf "%s-migrate" (include "goph-profile.fullname" .) }}
{{- end }}

{{/* OTEL-атрибуты ресурса с уникальным идентификатором pod. */}}
{{- define "goph-profile.otelResourceAttributes" -}}
{{- if .Values.config.otel.resourceAttributes -}}
{{- printf "%s,service.instance.id=$(POD_UID)" .Values.config.otel.resourceAttributes }}
{{- else -}}
service.instance.id=$(POD_UID)
{{- end }}
{{- end }}
