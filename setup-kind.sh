#!/bin/bash
# filepath: /home/simonb/workspace/goProjects/redokube/setup-kind.sh
set -e

# Définir des couleurs pour une meilleure lisibilité
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Configuration de l'opérateur Redokube dans un cluster Kind ===${NC}"

# Vérifier si kind est installé
if ! command -v kind &>/dev/null; then
  echo -e "${RED}Kind n'est pas installé. Veuillez l'installer avant de continuer.${NC}"
  echo "Pour installer Kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
  exit 1
fi

# Vérifier si docker est installé et en cours d'exécution
if ! docker info &>/dev/null; then
  echo -e "${RED}Docker n'est pas installé ou n'est pas en cours d'exécution. Veuillez installer Docker et le démarrer.${NC}"
  exit 1
fi

# Créer un cluster Kind s'il n'existe pas
if ! kind get clusters | grep -q "redokube-cluster"; then
  echo -e "${YELLOW}Création du cluster Kind 'redokube-cluster'...${NC}"
  kind create cluster --name redokube-cluster --config kind-config.yaml
  echo -e "${GREEN}Cluster Kind 'redokube-cluster' créé avec succès.${NC}"
else
  echo -e "${BLUE}Le cluster Kind 'redokube-cluster' existe déjà.${NC}"
fi

# S'assurer que kubectl est configuré pour utiliser le cluster kind
echo -e "${YELLOW}Configuration de kubectl pour utiliser le contexte du cluster Kind...${NC}"
kubectl cluster-info --context kind-redokube-cluster

# Compiler l'opérateur
echo -e "${YELLOW}Compilation de l'opérateur Redokube...${NC}"
go build -o bin/redokube ./cmd/
echo -e "${GREEN}Opérateur compilé avec succès.${NC}"

# Installer le CRD
echo -e "${YELLOW}Installation du CRD OpenAPISpec...${NC}"
kubectl apply -f config/crd.yaml
echo -e "${GREEN}CRD installé avec succès.${NC}"

# Construire l'image Docker
echo -e "${YELLOW}Construction de l'image Docker de l'opérateur...${NC}"
docker build -t redokube:latest .
echo -e "${GREEN}Image Docker construite avec succès.${NC}"

# Charger l'image dans le cluster Kind
echo -e "${YELLOW}Chargement de l'image dans le cluster Kind...${NC}"
kind load docker-image redokube:latest --name redokube-cluster
echo -e "${GREEN}Image chargée dans le cluster Kind avec succès.${NC}"

# Mettre à jour le fichier de déploiement pour utiliser l'image locale
echo -e "${YELLOW}Création d'un déploiement temporaire pour Kind...${NC}"
cat >deploy/kind-deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redokube
  labels:
    app: redokube
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redokube
  template:
    metadata:
      labels:
        app: redokube
    spec:
      serviceAccountName: redokube
      containers:
      - name: redokube
        image: redokube:latest
        imagePullPolicy: Never
        args:
        - "--health-probe-bind-address=:8081"
        - "--port=8082"
        - "--external-url=http://localhost:8082"
        - "--spec-directory=/data/specs"
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 8081
          name: health
        - containerPort: 8082
          name: docs
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        volumeMounts:
        - name: spec-storage
          mountPath: /data/specs
      volumes:
      - name: spec-storage
        emptyDir: {}
EOF

# Déployer les ressources nécessaires
echo -e "${YELLOW}Déploiement des composants de l'opérateur...${NC}"
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/kind-deployment.yaml
kubectl apply -f deploy/service.yaml
echo -e "${GREEN}Opérateur déployé avec succès.${NC}"

# Attendre que le déploiement soit prêt
echo -e "${YELLOW}Attente que le déploiement soit prêt...${NC}"
kubectl rollout status deployment/redokube
echo -e "${GREEN}Déploiement prêt.${NC}"

# Créer un exemple d'OpenAPISpec si le fichier exemple existe
if [ -f "examples/example-openapi-spec.yaml" ]; then
  echo -e "${YELLOW}Déploiement d'un exemple d'OpenAPISpec...${NC}"
  kubectl apply -f examples/example-openapi-spec.yaml
  echo -e "${GREEN}Exemple d'OpenAPISpec déployé.${NC}"
else
  echo -e "${YELLOW}Création d'un exemple d'OpenAPISpec...${NC}"
  mkdir -p examples
  cat >examples/example-openapi-spec.yaml <<EOF
apiVersion: docs.redokube.io/v1
kind: OpenAPISpec
metadata:
  name: petstore-api
spec:
  title: "Swagger Petstore"
  description: "This is a sample Petstore server."
  version: "1.0.0"
  specPath: "https://petstore3.swagger.io/api/v3/openapi.json"
EOF
  kubectl apply -f examples/example-openapi-spec.yaml
  echo -e "${GREEN}Exemple d'OpenAPISpec créé et déployé.${NC}"
fi

# Afficher les informations sur la façon d'accéder à l'interface utilisateur
echo -e "\n${GREEN}=========== DÉPLOIEMENT TERMINÉ ===========${NC}"
echo -e "${BLUE}L'interface de documentation OpenAPI devrait être accessible à l'adresse:${NC}"
echo -e "${GREEN}http://localhost:8082/${NC}"
echo -e "\n${BLUE}Pour vérifier l'état de votre ressource OpenAPISpec:${NC}"
echo -e "${YELLOW}kubectl get openapispecs${NC}"
echo -e "\n${BLUE}Pour voir les pods de l'opérateur:${NC}"
echo -e "${YELLOW}kubectl get pods${NC}"
echo -e "\n${BLUE}Pour accéder aux logs de l'opérateur:${NC}"
echo -e "${YELLOW}kubectl logs -l app=redokube${NC}"
echo -e "\n${BLUE}Pour supprimer le cluster Kind:${NC}"
echo -e "${YELLOW}kind delete cluster --name redokube-cluster${NC}"
