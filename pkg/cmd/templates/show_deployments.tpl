{{printf "%-40s%-15s%-15s%-15s" "DEPLOYMENT" "READY/DESIRED" "UP-TO-DATE" "VERSION" -}}
{{range $deployment := .items}}
{{printf "%-40s" $deployment.metadata.name -}}
{{printf "%2v/%-13v" (or $deployment.status.readyReplicas "-") (or $deployment.status.replicas "-") -}}
{{printf "%-15v" (or $deployment.status.updatedReplicas "-") -}}
{{printf "%-15v" (index $deployment.metadata.labels "app.kubernetes.io/version") -}}
{{end -}}