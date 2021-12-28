{{printf "%-70s%-15s%-50s" "NAME" "STATUS" "MESSAGE" -}}
{{range $postgresDump := .items }}
{{printf "%-70s" $postgresDump.metadata.name -}}
{{printf "%-15s" $postgresDump.status.status -}}
{{printf "%-50s" $postgresDump.status.message -}}
{{end -}} 