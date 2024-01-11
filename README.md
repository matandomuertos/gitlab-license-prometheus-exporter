

# Gitlab license Prometheus exporter
Exposes License expiration date and active users from the Gitlab API, to a Prometheus compatible endpoint.
Based on (gitlab_license_exporter)[https://github.com/jamf/gitlab_license_exporter] which was archived by the company who own it.

## Parameters
This exporter is setup to take input from these required environment variables:
* `TOKEN`: Gitlab Admin API token (License is only exposed to administrators)
* `URL`: Gitlab URL (ex. `https://gitlab.com`)

## Metrics
Metrics will be available on port 9090

| Name                      | Type  | Help                                                   |
| ------------------------- | ----- | ------------------------------------------------------ |
| gitlab_active_users       | gauge | Gitlab active users                                    |
| gitlab_license_expires_at | gauge | Gitlab expiration day                                  |
| gitlab_scrape_success     | gauge | Gitlab exporter scrape status when try to read the API |
| gitlab_user_limit         | gauge | Users allowed by license                               |

## Build and run
### Manually
```
go get
go build gitlabgoexporter.go
export TOKEN=token123token
export URL=https://gitlab.com
./gitlabgoexporter.go
```
Visit http://localhost:9090


### Docker
Build a docker image:
`docker build -t <image-name> .`

Run:
* Custom URL:
	`docker image --env TOKEN=token123token --env URL=https://gitlab.domain.com <image-name>`

* Kubernetes Gitlab-Web service:
	`docker image --env TOKEN=token123token <image-name>`

### Kubernetes
```
apiVersion: v1
kind: Secret
metadata:
  name: gitlab-token
  namespace: {{ NAMESPACE }}
  labels:
    app: gitlab-exporter
type: Opaque
data:
  token: {{ TOKEN | b64encode }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gitlab-exporter
  namespace: {{ NAMESPACE }}
  labels:
    app: gitlab-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gitlab-exporter
  template:
    metadata:
      labels:
        app: gitlab-exporter
    spec:
      containers:
      - name: gitlab-exporter
        image: {{{ image-name }}}
        ports:
        - containerPort: 2222
        env:
        - name: TOKEN
          valueFrom:
            secretKeyRef:
              name: gitlab-token
              key: token
        resources:
          limits:
            cpu: 100m
            memory: 128Mi
          requests:
            cpu: 50m
            memory: 64Mi
---
apiVersion: v1
kind: Service
metadata:
  name: gitlab-exporter-svc
  namespace: {{ NAMESPACE }}
  labels:
    app: gitlab-exporter
spec:
  selector: 
    app: gitlab-exporter
  ports:
    - name: metrics
      port: 8080
      targetPort: 2222
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: gitlab-export-metrics
  namespace: {{ NAMESPACE }}
spec:
  selector:
    matchLabels:
      app: gitlab-exporter
  endpoints:
  - port: metrics
    path: /
    interval: 30s
```
