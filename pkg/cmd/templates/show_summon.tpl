TENANT: {{ .metadata.name}}
STATE: {{.status.status}} ({{.status.message}})

DESIRED VERSIONS:
  {{- /* will need to update components manually */}}
  Summon: {{.spec.version}}
{{- if .spec.hwAux.version}}
  HwAux: {{.spec.hwAux.version}}{{end}}
{{- if .spec.dispatch.version}}
  Dispatch: {{.spec.dispatch.version}}{{end}}
{{- if .spec.businessPortal.version}}
  Business Portal: {{.spec.businessPortal.version}}{{end}}
{{- if .spec.pulse.version}}
  Pulse: {{.spec.pulse.version}}{{end}}
{{- if .spec.tripShare.version}}
  TripShare: {{.spec.tripShare.version}}{{end}}

CURRENT VERSIONS:
  {{range $key, $val := .status.notification}}{{if and (ne $key "slack") (ne $key "newRelic")}}{{$key}}: {{$val}}{{"\n  "}}{{end}}{{end}}
  Slack:
    {{range $key, $val := .status.notification.slack}}{{$key}}: {{$val}}{{"\n    "}}{{end}}
