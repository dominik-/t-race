# How-To: Establish GRPC-ingress on kubernetes cluster
## Requirements
- A kubernetes cluster (minikube works), with webhook Auth enabled and support for custom ingress.
- Helm installed, see https://docs.helm.sh/using_helm/#quickstart-guide
- Checkout this repo, so you have access to the YAML files for deployment. The assumption is that your working directory is the "deployment/kubernetes-single" folder.
## Step-by-Step Setup
1. (Optional if using minikube) Start minikube with parameters for webhook auth. You might consider giving the VM some extra memory and CPU cores.
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
1. Add configuration for different components to kubernetes
    ```bash
    kubectl create -f configmap.yml
    kubectl create -f configmap-benchmark.yml
    ```
1. Start cassandra backend: `kubectl create -f cassandra.yml`
1. WAIT for 2 minutes in order for the batch job to create the namespace on the cassandra cluster
1. Start jaeger backend: `kubectl create -f jaeger-backend.yml`
1. Start jaeger agent: `kubectl create -f jaeger-agend-daemon.yml`
1. Start a worker-writer pair: `kubectl create -f worker-writer.yml`
1. Setup nginx as ingress controller
    1. If using minikube, enable the plugin
    ```bash
    minikube addons enable ingress
    ```
    1. Otherwise, use the deployment from https://github.com/kubernetes/ingress-nginx/blob/master/docs/deploy/index.md
    ```bash
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/mandatory.yaml
    ```
    1. Create ingress config in Kubernetes
    ```bash
    kubectl create -f ingress.yml
    ```
1. Install prometheus via helm, configure access from outside.
    ```bash
    helm install coreos/prometheus-operator --name prometheus-operator --namespace monitoring
    helm install coreos/kube-prometheus --name kube-prometheus --set global.rbacEnable=true --namespace monitoring
    ```
    1. Modify prometheus services to type NodePort or use `kubectl expose` on deployments to create visibility from outside of cluster.

## Running benchmarks
1. Scale the number of workers in order to fit your deployment file.
