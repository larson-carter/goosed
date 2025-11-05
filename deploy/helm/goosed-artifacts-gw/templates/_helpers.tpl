{{- define "goosed-artifacts-gw.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "goosed-artifacts-gw.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "goosed-artifacts-gw.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "goosed-artifacts-gw.labels" -}}
app.kubernetes.io/name: {{ include "goosed-artifacts-gw.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "goosed-artifacts-gw.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goosed-artifacts-gw.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
