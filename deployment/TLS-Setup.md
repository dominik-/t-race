# How-To: Establish GRPC-ingress on kubernetes cluster
1. Start minikube with parameters for webhook auth. You might consider giving the VM some extra memory and CPU cores.
    ```bash
    minikube start --extra-config=kubelet.authentication-token-webhook=true --extra-config=kubelet.authorization-mode=Webhook
    ```
1. Generate a certificate for the frontend TLS termination at nginx.
    1. Generate cert, when prompted for "Common Name (e.g. server FQDN or YOUR name) []", enter `frontend.local`
    ```bash
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./certs/frontend.key -out ./certs/frontend.cert
    ```
1. Create kubernetes secrets, which contain the created certificates.
    ```bash
    kubectl create secret tls frontend-secret --key certs/frontend.key --cert certs/frontend.cert
    ```
1. Setup nginx as ingress controller
    1. If using minikube, enable the plugin
    ```bash
    minikube addons enable ingress
    ```
    1. Create ingress config in Kubernetes
    ```yaml
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
    name: worker-service-ingress
    annotations:
        kubernetes.io/ingress.class: "nginx"
        nginx.ingress.kubernetes.io/ssl-redirect: "true"
        nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
        nginx.ingress.kubernetes.io/grpc-backend: "true"
    spec:
    rules:
    - host: frontend.local
        http:
        paths:
        - backend:
            serviceName: worker-service
            servicePort: grpc
    tls:
    - secretName: frontend-secret
        hosts:
        - frontend.local
    ```
1. Install prometheus via helm, configure access from outside.
    ```bash
    helm install coreos/prometheus-operator --name prometheus-operator --namespace monitoring
    helm install coreos/kube-prometheus --name kube-prometheus --set global.rbacEnable=true --namespace monitoring
    ```
    1. Modify services to type NodePort or try to use expose on deployments to create visibility from outside of cluster.