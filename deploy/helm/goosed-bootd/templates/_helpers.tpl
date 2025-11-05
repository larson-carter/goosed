{{- define "goosed-bootd.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "goosed-bootd.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "goosed-bootd.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "goosed-bootd.labels" -}}
app.kubernetes.io/name: {{ include "goosed-bootd.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "goosed-bootd.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goosed-bootd.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
