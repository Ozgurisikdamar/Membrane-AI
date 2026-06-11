{{/*
Chart name, release-qualified fullname and shared labels. Service objects are
named "<fullname>-<service>" so several releases can coexist in a namespace.
*/}}

{{- define "membrane.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Truncated to 50, not 63: every object appends a suffix (longest
"-orchestrator" = 13, "-credentials" = 12), and that composed name is what
hits Kubernetes' 63-char limit. Capping the base here keeps suffixes intact
and distinct instead of letting them collapse into one truncated name.
*/}}
{{- define "membrane.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 50 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 50 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "membrane.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/name: {{ include "membrane.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Per-service object name. The composed "<fullname>-<service>" is what actually
hits Kubernetes' 63-char DNS-1035 limit, so truncate AFTER composing (the
fullname's own trunc never protects it). Used for both the object names and
the cross-service DNS in orchestrator env, so they can never diverge.
Expects a dict with "root" and "svcName".
*/}}
{{- define "membrane.svcFullname" -}}
{{- printf "%s-%s" (include "membrane.fullname" .root) .svcName | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Selector labels for one service; expects a dict with "root" and "svcName".
*/}}
{{- define "membrane.selectorLabels" -}}
app.kubernetes.io/name: {{ include "membrane.name" .root }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .svcName }}
{{- end -}}

{{/*
Image reference for one service; expects a dict with "root" and "svcName".
toString guards a numeric tag (CI date stamps) from printf's %!s(int64=…).
*/}}
{{- define "membrane.image" -}}
{{- $prefix := "" -}}
{{- if .root.Values.image.registry -}}
{{- $prefix = printf "%s/" .root.Values.image.registry -}}
{{- end -}}
{{- printf "%smembrane/%s:%s" $prefix .svcName (toString .root.Values.image.tag) -}}
{{- end -}}
