package utils

import "github.com/google/uuid"

// GenerateID creates a new unique identifier string.
func GenerateID() string {
	return uuid.NewString()
}
```
