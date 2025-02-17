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
	"solfa-back/models"
	"encoding/json"
	"bytes"
	"io"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

// Client Elasticsearch
var ESClient *elasticsearch.Client

var partition_index_name = "partitions_with_mapping"

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

// IndexPartitionInES indexe une partition dans Elasticsearch
func IndexPartitionInES(partition models.Partition) {
	// Convertir la partition en JSON
	jsonData, err := json.Marshal(partition)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"partition": partition,
			"error":     err.Error(),
		}).Error("Erreur lors de la sérialisation JSON de la partition")
		return
	}

	// Debug : Vérifier que le JSON est valide avant l'indexation
	logrus.WithFields(logrus.Fields{
		"jsonData": string(jsonData),
	}).Info("Données JSON à indexer dans Elasticsearch")

	// Indexer dans Elasticsearch avec un buffer bytes.NewReader()
	res, err := ESClient.Index(
		partition_index_name,                   // Nom de l'index
		bytes.NewReader(jsonData),       // Envoyer les données correctement formatées
	)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"partition": partition,
			"error":     err.Error(),
		}).Error("Erreur lors de l'indexation dans Elasticsearch")
		return
	}
	defer res.Body.Close()

	// Vérification du statut de la réponse Elasticsearch
	if res.IsError() {
		body, _ := io.ReadAll(res.Body) // Lire la réponse pour un meilleur débogage
		logrus.WithFields(logrus.Fields{
			"partition": partition,
			"status":    res.Status(),
			"response":  string(body),
		}).Error("Elasticsearch a renvoyé une erreur lors de l'indexation")
		return
	}

	// Log succès
	logrus.WithFields(logrus.Fields{
		"partition_id": partition.ID, // Supposant que Partition a un champ ID
		"status":       "success",
	}).Info("Partition indexée avec succès dans Elasticsearch")
}


func generateHash(partition models.Partition) string {
	// Concaténer les champs de la partition
	data := fmt.Sprintf("%s%s%s", partition.Title, partition.Composer, partition.Genre, partition.Category) 
	// Générer le hash MD5
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func SearchPartitionByHash(partition models.Partition) (bool, interface{}) {
	// Générer le hash de la partition
	partitionHash := generateHash(partition)

	// Rechercher si ce hash existe déjà dans Elasticsearch
	res, err := ESClient.Search(
		ESClient.Search.WithIndex(partition_index_name),
		ESClient.Search.WithBody(strings.NewReader(fmt.Sprintf(`
			{
				"query": {
					"match": {
						"partition_hash": "%s"
					}
				}
			}`, partitionHash))),
	)
	if err != nil {
		fmt.Println("Erreur lors de la recherche dans Elasticsearch:", err)
		return false, nil
	}
	defer res.Body.Close()

	// Analyser la réponse pour voir si la partition existe déjà
	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)
	hits := result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"]
	if hits.(float64) > 0 {
		fmt.Println("Partition similaire déjà présente.")
		return true, result["hits"].(map[string]interface{})["hits"].([]interface{})[0]
	}

	return false, nil
}

func SearchPartitionByFields(partition models.Partition) (bool, interface{}) {
	// Rechercher sur plusieurs champs de la partition
	res, err := ESClient.Search(
		ESClient.Search.WithIndex(partition_index_name),
		ESClient.Search.WithBody(strings.NewReader(fmt.Sprintf(`
			{
				"query": {
					"multi_match": {
						"query": "%s",
						"fields": ["title", "composer", "genre", "category"],
						"operator": "and",
						"fuzziness": "AUTO"
					}
				}
			}`, partition.Title)), // Ici tu peux inclure un autre champ (par exemple partition.Title)
	))
	if err != nil {
		fmt.Println("Erreur lors de la recherche dans Elasticsearch:", err)
		return false, nil
	}
	defer res.Body.Close()

	// Analyser la réponse pour voir si la partition existe déjà
	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)
	hits := result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"]
	if hits.(float64) > 0 {
		fmt.Println("Partition similaire déjà présente.")
		return true, result["hits"].(map[string]interface{})["hits"].([]interface{})[0]
	}

	return false, nil
}
