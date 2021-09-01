apiVersion: app.summon.ridecell.io/v1beta2
kind: SummonPlatform
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  version: fill_in_version
  {{- if .SlackChannels }}
  notifications:
    slackChannels: 
    {{- range .SlackChannels }}
    - "{{ . }}"
    {{- end }}
  {{- end }}
---
apiVersion: secrets.controllers.ridecell.io/v1beta2
kind: EncryptedSecret
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
data: {}