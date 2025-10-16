{{/*
Expand the name of the chart.
*/}}
{{- define "bib-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "bib-operator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{ include "bib-operator.name" . }}
{{- end -}}
{{- end }}

{{/*
Service account name.
*/}}
{{- define "bib-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.name -}}
{{- .Values.serviceAccount.name -}}
{{- else -}}
{{ include "bib-operator.fullname" . }}
{{- end -}}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "bib-operator.labels" -}}
helm.sh/chart: {{ include "bib-operator.chart" . }}
app.kubernetes.io/name: {{ include "bib-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "bib-operator.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end }}
