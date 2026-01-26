{{/*
Expand the name of the chart.
*/}}
{{- define "philotes-api.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "philotes-api.fullname" -}}
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
{{- define "philotes-api.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "philotes-api.labels" -}}
helm.sh/chart: {{ include "philotes-api.chart" . }}
{{ include "philotes-api.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: philotes
{{- end }}

{{/*
Selector labels
*/}}
{{- define "philotes-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "philotes-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: api
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "philotes-api.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "philotes-api.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the database password secret name
*/}}
{{- define "philotes-api.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- include "philotes-api.fullname" . }}-db
{{- end }}
{{- end }}

{{/*
Return true if a database secret should be created
*/}}
{{- define "philotes-api.createDatabaseSecret" -}}
{{- if and (not .Values.database.existingSecret) .Values.database.password }}
{{- true }}
{{- end }}
{{- end }}
