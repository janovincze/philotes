{{/*
Expand the name of the chart.
*/}}
{{- define "philotes-worker.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "philotes-worker.fullname" -}}
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
{{- define "philotes-worker.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "philotes-worker.labels" -}}
helm.sh/chart: {{ include "philotes-worker.chart" . }}
{{ include "philotes-worker.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: philotes
{{- end }}

{{/*
Selector labels
*/}}
{{- define "philotes-worker.selectorLabels" -}}
app.kubernetes.io/name: {{ include "philotes-worker.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: worker
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "philotes-worker.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "philotes-worker.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the database password secret name
*/}}
{{- define "philotes-worker.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- include "philotes-worker.fullname" . }}-db
{{- end }}
{{- end }}

{{/*
Get the source database password secret name
*/}}
{{- define "philotes-worker.sourceSecretName" -}}
{{- if .Values.source.existingSecret }}
{{- .Values.source.existingSecret }}
{{- else }}
{{- include "philotes-worker.fullname" . }}-source
{{- end }}
{{- end }}

{{/*
Get the storage secret name
*/}}
{{- define "philotes-worker.storageSecretName" -}}
{{- if .Values.storage.existingSecret }}
{{- .Values.storage.existingSecret }}
{{- else }}
{{- include "philotes-worker.fullname" . }}-storage
{{- end }}
{{- end }}

{{/*
Return true if secrets should be created
*/}}
{{- define "philotes-worker.createSecrets" -}}
{{- if or (and (not .Values.database.existingSecret) .Values.database.password) (and (not .Values.source.existingSecret) .Values.source.password) (and (not .Values.storage.existingSecret) .Values.storage.accessKey) }}
{{- true }}
{{- end }}
{{- end }}
