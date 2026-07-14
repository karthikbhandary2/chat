package presence

import (
	"fmt"
)

func Key(userID string) string {
	return fmt.Sprintf("presence:%s", userID)
}
