apiVersion: summon.ridecell.io/v1beta1
kind: SummonPlatform
metadata:
  name: {{ .Name }}-{{ .Namespace }}
  namespace: {{ .Namespace }}
spec:
  version: fill_in_version
---
apiversion: secrets.ridecell.io/v1beta1
kind: EncryptedSecret
metadata:
  name: {{ .Name }}-{{ .Namespace }}
  namespace: {{ .Namespace }}
data: {}
