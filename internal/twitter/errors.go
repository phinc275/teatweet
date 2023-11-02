package twitter

import (
	"fmt"
	"time"
)

type rateLimitError struct {
	Reset int64
}

func (err *rateLimitError) Error() string {
	return fmt.Sprintf("retry after %s", time.Unix(err.Reset, 0))
}
