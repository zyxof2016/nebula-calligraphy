{{- define "nebula-calligraphy.name" -}}
nebula-calligraphy
{{- end }}

{{- define "nebula-calligraphy.labels" -}}
app.kubernetes.io/name: {{ include "nebula-calligraphy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "nebula-calligraphy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nebula-calligraphy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "nebula-calligraphy.image" -}}
{{- $digest := .Values.image.digest | default "" -}}
{{- if and .Values.production.requireImageDigest (or (not (regexMatch "^sha256:[a-f0-9]{64}$" $digest)) (eq $digest "sha256:0000000000000000000000000000000000000000000000000000000000000000")) -}}
{{- fail "production requires image.digest to be a non-placeholder sha256 digest" -}}
{{- end -}}
{{- if $digest -}}
{{- printf "%s@%s" .Values.image.repository $digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository .Values.image.tag -}}
{{- end -}}
{{- end }}
