# How-To: Establish GRPC-ingress on kubernetes cluster
1. Generate two certificates, one for backend GRPC Server, one for frontend TLS termination at traefik. Taken from: https://docs.traefik.io/user-guide/grpc/
    1. Generate backend cert, when prompted for "Common Name (e.g. server FQDN or YOUR name) []", enter `backend.local`
    ```bash
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./certs/backend.key -out ./certs/backend.cert
    ```
    2. Same for frontend certs, on prompt for host name enter `frontend.local`
    ```bash
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./certs/frontend.key -out ./certs/frontend.cert
    ```
1. Create kubernetes secrets, which contain the created certificates.
    ```bash
    kubectl create secret tls backend-secret --key certs/backend.key --cert certs/backend.cert
    kubectl -n kube-system create secret tls frontend-secret --key certs/frontend.key --cert certs/frontend.cert
    ```
1. Setup traefik as ingress controller
    1. Create traefik cluster role
    ```bash
    kubectl apply -f https://raw.githubusercontent.com/containous/traefik/master/examples/k8s/traefik-rbac.yaml
    ```
    1. Create config.toml for traefik as kubernetes ConfigMap resource
    1. Start traefik daemon process