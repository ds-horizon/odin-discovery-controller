{{/*
Create the name of the service account to use
*/}}
{{- define "odin-service-discovery-controller.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "common.names.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate imagePullSecrets list using Bitnami common helper
Uses image.pullSecrets from the image configuration
*/}}
{{- define "odin-service-discovery-controller.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.image) "global" (dict "imagePullSecrets" list)) }}
{{- end }}
