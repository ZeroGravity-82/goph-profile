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

{{/* Имя Secret с TLS-сертификатом основного HTTP-сервера. */}}
{{- define "goph-profile.tlsSecretName" -}}
{{- printf "%s-server-tls" (include "goph-profile.fullname" .) }}
{{- end }}
