package adoption

import (
	"strconv"
)

// strconvItoa wraps strconv.Itoa so rule files don't each need the import.
func strconvItoa(n int) string { return strconv.Itoa(n) }
