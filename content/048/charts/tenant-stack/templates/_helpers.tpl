{{/*
Short labels shared by every resource.
*/}}
{{- define "tenant-stack.labels" -}}
app.kubernetes.io/managed-by: obs-saas
obs-saas.io/tenant: {{ .Values.tenant.id | quote }}
obs-saas.io/chart-version: {{ .Chart.Version | quote }}
{{- end -}}

{{- define "tenant-stack.host" -}}
{{- $svc := .svc -}}
{{- printf "%s-%s.%s" .Values.tenant.id $svc .Values.tenant.ingestDomain -}}
{{- end -}}

{{- define "tenant-stack.require" -}}
{{- if not .Values.tenant.id }}{{- fail "tenant.id is required" }}{{- end -}}
{{- if not .Values.tenant.grafanaAdminPassword }}{{- fail "tenant.grafanaAdminPassword is required" }}{{- end -}}
{{- if not .Values.tenant.ingestJWT.secret }}{{- fail "tenant.ingestJWT.secret is required" }}{{- end -}}
{{- if not .Values.tenant.ingestJWT.expectedTid }}{{- fail "tenant.ingestJWT.expectedTid is required" }}{{- end -}}
{{- if not .Values.clickhouse.password }}{{- fail "clickhouse.password is required" }}{{- end -}}
{{- end -}}
