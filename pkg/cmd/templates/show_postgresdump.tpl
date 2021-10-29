{{range $deployment := .items }}
NAME: {{$deployment.metadata.name}}
STATUS: {{$deployment.status.status}}
MESSAGE: {{$deployment.status.message}}
{{if $deployment.status.dumpPath  }}
S3PATH: {{$deployment.status.dumpPath }}
{{end}}
{{end -}}