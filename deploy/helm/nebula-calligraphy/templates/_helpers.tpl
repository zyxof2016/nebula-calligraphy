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
