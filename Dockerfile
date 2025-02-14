# Étape 1 : Build de l'application
FROM golang:1.21 AS builder

# Définir le répertoire de travail dans le conteneur
WORKDIR /app

# Copier les fichiers go et les fichiers de dépendances
COPY go.mod go.sum ./
RUN go mod download

# Copier le reste du code source
COPY . .

# Compiler l'application en binaire exécutable
RUN go build -o solfa-api .

# Étape 2 : Création de l'image finale minimaliste
FROM alpine:latest

# Définir un répertoire de travail
WORKDIR /root/

# Copier le binaire depuis le builder
COPY --from=builder /app/solfa-api .

# Exposer le port sur lequel l’API écoute
EXPOSE 8080

# Démarrer l’API
CMD ["./solfa-api"]
