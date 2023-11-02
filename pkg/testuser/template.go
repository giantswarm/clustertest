package testuser

type templateVars struct {
	CA          string
	Endpoint    string
	AccountName string
	Token       string
}

var kubeConfigTemplate = `apiVersion: v1
kind: Config
clusters:
  - name: e2e-wc
    cluster:
      certificate-authority-data: {{ .CA }}
      server: {{ .Endpoint }}
contexts:
  - name: {{ .AccountName }}@e2e-wc
    context:
      cluster: e2e-wc
      namespace: default
      user: {{ .AccountName }}
users:
  - name: {{ .AccountName }}
    user:
      token: {{ .Token }}
current-context: {{ .AccountName }}@e2e-wc
`
