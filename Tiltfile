allow_k8s_contexts('minikube')

# Both images are pre-built and loaded into minikube:
# - reverxy: docker build -t reverxy:latest . && minikube image load reverxy:latest
# - reverxy-backend: docker build -t reverxy-backend:latest -f Dockerfile.backend . && minikube image load reverxy-backend:latest
# Tilt will use the existing images with imagePullPolicy: IfNotPresent

# Apply k8s manifests from files
k8s_yaml([
    'k8s/base/configmap.yaml',
    'k8s/base/backends.yaml',
    'k8s/base/proxy.yaml',
])

# Port forwards for easy local access
k8s_resource('reverxy', port_forwards=['8080:8080', '8085:8085'])
k8s_resource('backend-1', port_forwards=['8081:8081'])
k8s_resource('backend-2', port_forwards=['8082:8082'])
k8s_resource('backend-3', port_forwards=['8083:8083'])