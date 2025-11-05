{{- define "goosed-observability.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "goosed-observability.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s" $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "goosed-observability.labels" -}}
helm.sh/chart: {{ include "goosed-observability.chart" . }}
app.kubernetes.io/name: {{ include "goosed-observability.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "goosed-observability.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goosed-observability.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "goosed-observability.chart" -}}
{{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
