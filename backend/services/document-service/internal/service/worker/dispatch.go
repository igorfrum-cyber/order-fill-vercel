package worker

import "fmt"

const (
	TypeOrderFill  = "order_fill"
	TypeNorthMerge = "north_merge"
)

func Dispatch(jobType string) (string, error) {
	switch jobType {
	case TypeOrderFill:
		return "orderfill", nil
	case TypeNorthMerge:
		return "north", nil
	default:
		return "", fmt.Errorf("unknown job type %q", jobType)
	}
}
