package snapshot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
)

var ttlMatch = regexp.MustCompile(`^\d+(s|m|h)$`)

func ParseTTL(s string) (*snapshottypes.ParsedTTL, error) {
	parsedTTLResponse := &snapshottypes.ParsedTTL{}

	matches := ttlMatch.FindStringSubmatch(s)
	if len(matches) < 2 {
		return nil, errors.New("Failed to get a valid match")
	}

	unit := matches[1]
	quantity := strings.Split(ttlMatch.FindStringSubmatch(s)[0], unit)
	quantityInt, err := strconv.ParseInt(quantity[0], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parseInt quanitity")
	}

	switch unit {
	case "s":
		parsedTTLResponse.Quantity = quantityInt
		parsedTTLResponse.Unit = "seconds"
		break
	case "m":
		parsedTTLResponse.Quantity = quantityInt
		parsedTTLResponse.Unit = "minutes"
		break
	case "h":
		if quantityInt/8766 >= 1 && quantityInt%8766 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 8766
			parsedTTLResponse.Unit = "years"
			break
		}
		if quantityInt/720 >= 1 && quantityInt%720 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 720
			parsedTTLResponse.Unit = "months"
			break
		}
		if quantityInt/168 >= 1 && quantityInt%168 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 168
			parsedTTLResponse.Unit = "weeks"
			break
		}
		if quantityInt/24 >= 1 && quantityInt%24 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 24
			parsedTTLResponse.Unit = "days"
			break
		}
		parsedTTLResponse.Quantity = quantityInt
		parsedTTLResponse.Unit = "hours"
		break
	default:
		return nil, fmt.Errorf("unsupported unit type")
	}
	return parsedTTLResponse, nil
}

func FormatTTL(quantity string, unit string) (string, error) {
	n, err := strconv.Atoi(quantity)
	if err != nil {
		return "", err
	}

	switch unit {
	case "seconds":
		return fmt.Sprintf("%ds", n), nil
	case "minutes":
		return fmt.Sprintf("%dm", n), nil
	case "hours":
		return fmt.Sprintf("%dh", n), nil
	case "days":
		return fmt.Sprintf("%dh", n*24), nil
	case "weeks":
		return fmt.Sprintf("%dh", n*168), nil
	case "months":
		return fmt.Sprintf("%dh", n*720), nil
	case "years":
		return fmt.Sprintf("%dh", n*8766), nil
	}

	return "", fmt.Errorf("Invalid snapshot TTL: %d %s", n, unit)
}
