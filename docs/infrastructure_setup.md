# Local Production Infrastructure Setup Guide

This guide describes how to simulate a production environment locally using **Argo CD** for Continuous Delivery and **Jenkins** for Continuous Integration on a Kubernetes cluster (Minikube, Kind, or Docker Desktop).

## Prerequisites
-   A running Kubernetes cluster (`minikube start` or Docker Desktop K8s enabled).
-   `kubectl` installed and configured.
-   `git` installed.

## 1. Install Argo CD
Since we are not using Helm, we install Argo CD using the official manifests.

```bash
# Create namespace
kubectl create namespace argocd

# Apply official installation manifest
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for pods to be ready
kubectl wait --for=condition=Ready pods --all -n argocd --timeout=300s
```

### Access Argo CD UI
In a separate terminal, port-forward the Argo CD server:
```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```
-   **URL**: `https://localhost:8080`
-   **Username**: `admin`
-   **Password**: Get the initial password:
    ```bash
    kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo
    ```

## 2. Deploy Connectify Application (Argo CD)
Apply the Application CRD to tell Argo CD to sync our `k8s/` directory.

> [!IMPORTANT]
> Ensure the `repoURL` in [k8s/platform/argocd/connectify-app.yaml](file:///Users/a.k.mmuhibullahnayem/Developer/connectify-v2/k8s/platform/argocd/connectify-app.yaml) points to your actual Git repository where these files are pushed.

```bash
kubectl apply -f k8s/platform/argocd/connectify-app.yaml
```

## 3. Install Jenkins
Deploy Jenkins to the `default` namespace.

```bash
kubectl apply -f k8s/platform/jenkins/rbac.yaml
kubectl apply -f k8s/platform/jenkins/deployment.yaml
kubectl apply -f k8s/platform/jenkins/service.yaml
```

### Access Jenkins UI
Jenkins is exposed via NodePort `30000`. If using Minikube, you might need `minikube service jenkins --url`.
Alternatively, port-forward:
```bash
kubectl port-forward svc/jenkins 8081:8080
```
-   **URL**: `http://localhost:8081`
-   **Unlock Jenkins**: Retrieve the initial admin password from logs:
    ```bash
    kubectl logs -l app=jenkins
    ```

## 4. Verification
1.  **Argo CD**: Check the `connectify-v2` application in the UI. It should be "Synced" and "Healthy".
2.  **Jenkins**: Log in and verify you can create new Jobs.
3.  **Deployments**: `kubectl get pods` should show `messaging-app`, `user-service`, etc., running (synced by Argo CD).
