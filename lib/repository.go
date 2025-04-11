package lib

import (
	"solfa-back/models"
	"gorm.io/gorm"
)

func GetUserByUsername(username string) (*models.User, error) {
    var user models.User

    // Rechercher l'utilisateur par nom d'utilisateur
    if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil // Retourne nil si l'utilisateur n'existe pas
        }
        return nil, err // Retourne nil et l'erreur pour les autres cas
    }

    return &user, nil // Retourne un pointeur vers l'utilisateur trouvé
}