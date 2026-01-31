{{/*
Expand the name of the chart.
*/}}
{{- define "trino.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "trino.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "trino.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "trino.labels" -}}
helm.sh/chart: {{ include "trino.chart" . }}
{{ include "trino.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "trino.selectorLabels" -}}
app.kubernetes.io/name: {{ include "trino.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Coordinator selector labels
*/}}
{{- define "trino.coordinator.selectorLabels" -}}
{{ include "trino.selectorLabels" . }}
app.kubernetes.io/component: coordinator
{{- end }}

{{/*
Worker selector labels
*/}}
{{- define "trino.worker.selectorLabels" -}}
{{ include "trino.selectorLabels" . }}
app.kubernetes.io/component: worker
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "trino.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "trino.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Coordinator name
*/}}
{{- define "trino.coordinator.name" -}}
{{- printf "%s-coordinator" (include "trino.fullname" .) }}
{{- end }}

{{/*
Worker name
*/}}
{{- define "trino.worker.name" -}}
{{- printf "%s-worker" (include "trino.fullname" .) }}
{{- end }}

{{/*
ConfigMap name for Trino configuration
*/}}
{{- define "trino.configmap.name" -}}
{{- printf "%s-config" (include "trino.fullname" .) }}
{{- end }}

{{/*
ConfigMap name for Trino catalogs
*/}}
{{- define "trino.catalogs.configmap.name" -}}
{{- printf "%s-catalogs" (include "trino.fullname" .) }}
{{- end }}

{{/*
Discovery URI for Trino
*/}}
{{- define "trino.discoveryUri" -}}
{{- printf "http://%s:%d" (include "trino.coordinator.name" .) (.Values.service.port | int) }}
{{- end }}
