{{/*
Expand the name of the chart.
*/}}
{{- define "lakekeeper.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "lakekeeper.fullname" -}}
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
{{- define "lakekeeper.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "lakekeeper.labels" -}}
helm.sh/chart: {{ include "lakekeeper.chart" . }}
{{ include "lakekeeper.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: philotes
{{- end }}

{{/*
Selector labels
*/}}
{{- define "lakekeeper.selectorLabels" -}}
app.kubernetes.io/name: {{ include "lakekeeper.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: catalog
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "lakekeeper.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "lakekeeper.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the database secret name
*/}}
{{- define "lakekeeper.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- include "lakekeeper.fullname" . }}-db
{{- end }}
{{- end }}

{{/*
Build the database URL
*/}}
{{- define "lakekeeper.databaseUrl" -}}
postgresql://{{ .Values.database.user }}:$(LAKEKEEPER_DB_PASSWORD)@{{ .Values.database.host }}:{{ .Values.database.port }}/{{ .Values.database.name }}
{{- end }}
