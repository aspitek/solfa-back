# Étape 1 : Build de l'application
FROM golang:1.21 AS builder

WORKDIR /app

# Copier uniquement go.mod (et go.sum s'il existe)
COPY go.mod ./
RUN go mod tidy  # Assure la génération de go.sum

# Copier le fichier go.sum généré
COPY go.sum ./

# Télécharger les dépendances
RUN go mod download

# Copier le reste du code source
COPY . .

# Compiler l'application en binaire exécutable
RUN go build -o solfa-api .

# Étape 2 : Création de l'image finale minimaliste
FROM alpine:latest

WORKDIR /root/

# Installer les dépendances nécessaires pour exécuter l'application
RUN apk add --no-cache ca-certificates

# Copier le binaire depuis le builder
COPY --from=builder /app/solfa-api .

# Exposer le port sur lequel l’API écoute
EXPOSE 8080

# Démarrer l’API
CMD ["./solfa-api"]
