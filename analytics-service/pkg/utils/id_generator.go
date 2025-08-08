package utils

import (
	"fmt"

	"github.com/google/uuid"
)

func GenerateID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String())
}
