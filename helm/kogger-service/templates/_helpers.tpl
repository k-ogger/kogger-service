{{- define "kogger-service.name" -}}
{{- default "kogger-service" .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Helm required labels */}}
{{- define "kogger-service.labels" -}}
heritage: {{ .Release.Service }}
release: {{ .Release.Name }}
chart: {{ .Chart.Name }}
app: "{{ template "kogger-service.name" . }}"
{{- end -}}

{{/* matchLabels */}}
{{- define "kogger-service.matchLabels" -}}
release: {{ .Release.Name }}
app: "{{ template "kogger-service.name" . }}"
{{- end -}}