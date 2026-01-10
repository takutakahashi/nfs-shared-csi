{{/*
Expand the name of the chart.
*/}}
{{- define "nfs-shared-csi.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "nfs-shared-csi.fullname" -}}
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
{{- define "nfs-shared-csi.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nfs-shared-csi.labels" -}}
helm.sh/chart: {{ include "nfs-shared-csi.chart" . }}
{{ include "nfs-shared-csi.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nfs-shared-csi.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nfs-shared-csi.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Node selector labels
*/}}
{{- define "nfs-shared-csi.nodeSelectorLabels" -}}
app: csi-nfs-node
{{ include "nfs-shared-csi.selectorLabels" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "nfs-shared-csi.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (printf "%s-node-sa" (include "nfs-shared-csi.fullname" .)) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
CSI driver name
*/}}
{{- define "nfs-shared-csi.driverName" -}}
{{- .Values.driver.name }}
{{- end }}

{{/*
Driver image
*/}}
{{- define "nfs-shared-csi.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Kubelet plugin path
*/}}
{{- define "nfs-shared-csi.kubeletPluginPath" -}}
{{- printf "%s/plugins/%s" .Values.kubelet.dir .Values.driver.name }}
{{- end }}

{{/*
Kubelet registration path
*/}}
{{- define "nfs-shared-csi.kubeletRegistrationPath" -}}
{{- printf "%s/plugins/%s/csi.sock" .Values.kubelet.dir .Values.driver.name }}
{{- end }}
