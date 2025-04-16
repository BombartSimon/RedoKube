# OpenAPI Operator

Un opérateur Kubernetes pour centraliser et afficher la documentation OpenAPI des APIs.

## Description

L'OpenAPI Operator est un opérateur Kubernetes développé en Go qui permet de centraliser la documentation des APIs en format OpenAPI/Swagger et de les afficher via une interface Redoc. Cet opérateur scanne et gère des ressources personnalisées OpenAPISpec qui pointent vers des fichiers de spécification OpenAPI, et génère automatiquement une documentation interactive accessible depuis le navigateur.

## Fonctionnalités

- Déploiement simple sur Kubernetes
- Gestion de multiples spécifications OpenAPI
- Mise à jour automatique de la documentation lors des changements de spécification
- Interface utilisateur Redoc intégrée pour une documentation moderne et interactive
- Surveillance de l'état des spécifications via les status Kubernetes
- Support pour les fichiers de spécification locaux ou distants (URL)

## Prérequis

- Kubernetes 1.19+
- Kubectl 1.19+
- Go 1.21+ (pour le développement)
- Docker (pour la construction des images)

## Installation

### Installation des CRD

```bash
make install-crd
```

### Déploiement de l'opérateur

1. Modifiez la variable `REGISTRY` dans le Makefile pour pointer vers votre registry Docker
2. Construisez et publiez l'image Docker :

```bash
make docker-build
make docker-push
```

3. Déployez l'opérateur sur votre cluster :

```bash
make deploy
```

## Utilisation

Une fois l'opérateur déployé, vous pouvez créer des ressources OpenAPISpec pour afficher la documentation de vos APIs :

```yaml
apiVersion: docs.redokube.io/v1
kind: OpenAPISpec
metadata:
  name: ma-super-api
spec:
  title: "Ma Super API"
  description: "Documentation de mon API"
  version: "1.0.0"
  specPath: "https://chemin-vers-mon-fichier-openapi.json"
```

Appliquez ce fichier à votre cluster :

```bash
kubectl apply -f ma-super-api.yaml
```

L'opérateur détectera automatiquement cette ressource, traitera le fichier OpenAPI spécifié, et mettra à jour le statut avec l'URL où la documentation est accessible.

Pour vérifier l'état de votre documentation :

```bash
kubectl get openapispecs
```

## Exemple

Un exemple de spécification est disponible dans le dossier `examples/` :

```bash
make example
```

## Architecture

L'opérateur est composé des éléments suivants :

- **Custom Resource Definition (CRD)** : Define le type de ressource OpenAPISpec
- **Controller** : Surveille les ressources OpenAPISpec et réconcilie leur état
- **Redoc Server** : Serveur web qui héberge les documentations générées

## Développement

### Exécution locale

Pour exécuter l'opérateur localement pendant le développement :

```bash
make run
```

### Tests

Pour exécuter les tests unitaires :

```bash
make test
```

## Licence

Ce projet est sous licence MIT.