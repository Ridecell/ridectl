TENANT: {{ .metadata.name}}
STATE: {{.status.status}} ({{.status.message}})

  Postgres: {{.status.postgresStatus}}
    host: {{.status.postgresConnection.host}}
    database: {{.status.postgresConnection.database}}
    username: {{.status.postgresConnection.username}}

  RabbitMQ: {{.status.rabbitmqStatus}}
    host: {{.status.rabbitmqConnection.host}}

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
  Summon: {{.status.notification.notifyVersion}}
  {{range $key, $val := .status.notification}}{{if ne $key "notifyVersion"}}{{$key}}: {{$val}}{{"\n  "}}{{end}}{{end}}