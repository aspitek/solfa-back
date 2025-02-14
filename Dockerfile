# Étape 1 : Build de l'application
FROM golang:1.21 AS builder

WORKDIR /app

# Copier uniquement go.mod et go.sum
COPY go.mod ./
RUN go mod tidy  # Assure la génération de go.sum
COPY go.sum ./

# Télécharger les dépendances
RUN go mod download

# Copier le reste du code source
COPY . .

# Compiler l'application en binaire exécutable
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o solfa-api .
RUN chmod +x solfa-api
RUN ls -l /app/solfa-api  # Vérification

# Étape 2 : Création de l'image finale minimaliste
FROM alpine:latest

WORKDIR /root/

# Installer les dépendances nécessaires pour exécuter l'application
RUN apk add --no-cache ca-certificates

# Copier le binaire depuis le builder
COPY --from=builder /app/solfa-api .
RUN ls -l /root/solfa-api  # Vérification

# Exposer le port sur lequel l’API écoute
EXPOSE 8080

# Démarrer l’API
CMD ["./solfa-api"]
