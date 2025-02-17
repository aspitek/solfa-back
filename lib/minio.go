package lib
import (
	"os"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"fmt"
)

var MinioClient *minio.Client

// Initialisation du client Minio (à faire dans un fichier de configuration)
func InitMC() {
	var err error
	MinioClient, err = minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), ""),
		Secure: false,
	})
	if err != nil {
		fmt.Println("Erreur lors de la connexion à Minio", err)
	}
}
