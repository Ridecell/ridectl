STATUS: {{.status.status  }}
MESSAGE: {{.status.message -}}
{{if .status.dumpPath  }}
S3PATH: {{.status.dumpPath }}{{end}}