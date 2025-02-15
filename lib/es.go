package lib

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/sirupsen/logrus"
	"log"
	"time"
	"os"
	"crypto/tls"
	"net/http"
)

// Client Elasticsearch
var ESClient *elasticsearch.Client

// Initialisation de la connexion à Elasticsearch
func InitES() {

	// Configurer la connexion à Elasticsearch avec les identifiants et l'URL
	tlsConfig := &tls.Config{
        InsecureSkipVerify: true,  // Désactive la vérification du certificat
    }

	cfg := elasticsearch.Config{
		Addresses: []string{
			os.Getenv("ES_HOST"), // Remplace par ton adresse Elasticsearch
		},
		Username: os.Getenv(("ES_USER")),  // Utilisateur chargé depuis .env
		Password: os.Getenv("ES_PASSWORD"),  // Mot de passe chargé depuis .env
		Transport: &http.Transport{TLSClientConfig: tlsConfig},  // Applique la config TLS ici
	}

	var err error
	ESClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Erreur lors de la connexion à Elasticsearch: %s", err)
	}
}

// LogAction enregistre une action utilisateur dans Elasticsearch
func LogAction(action string, email string) {
	logData := map[string]interface{}{
		"action":    action,
		"email":     email,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Indexer dans Elasticsearch sans spécifier de type de document
	res, err := ESClient.Index(
		"user_actions", // Nom de l'index
		esutil.NewJSONReader(logData), // Sérialisation des données en JSON
	)
	if err != nil {
		// Gestion des erreurs lors de l'indexation dans Elasticsearch
		logrus.WithFields(logrus.Fields{
			"action":   action,
			"email":    email,
			"error":    err.Error(),
		}).Error("Erreur lors de l'enregistrement de l'action dans Elasticsearch")
		return
	}
	defer res.Body.Close()

	// Log dans la console pour le suivi
	logrus.WithFields(logrus.Fields{
		"action":  action,
		"email":   email,
		"status":  "success",
	}).Info("Action enregistrée dans Elasticsearch")
}
