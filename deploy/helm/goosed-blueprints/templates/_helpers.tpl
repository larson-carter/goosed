{{- define "goosed-blueprints.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "goosed-blueprints.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "goosed-blueprints.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "goosed-blueprints.labels" -}}
app.kubernetes.io/name: {{ include "goosed-blueprints.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "goosed-blueprints.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goosed-blueprints.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
