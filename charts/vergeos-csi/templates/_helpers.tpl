{{- define "vergeos-csi.name" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "vergeos-csi.secretName" -}}
{{- if .Values.vergeos.existingSecret -}}
{{- .Values.vergeos.existingSecret -}}
{{- else -}}
{{- include "vergeos-csi.name" . -}}-credentials
{{- end -}}
{{- end -}}

{{- define "vergeos-csi.image" -}}
{{ .Values.image.repository }}:{{ .Values.image.tag }}
{{- end -}}
