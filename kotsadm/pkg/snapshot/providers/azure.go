package providers

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type Azure struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	ResourceGroup  string
	CloudName      string
}

func ParseAzureConfig(data []byte) Azure {
	var config Azure

	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "=")
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "AZURE_SUBSCRIPTION_ID":
			config.SubscriptionID = val
		case "AZURE_TENANT_ID":
			config.TenantID = val
		case "AZURE_CLIENT_ID":
			config.ClientID = val
		case "AZURE_CLIENT_SECRET":
			config.ClientSecret = val
		case "AZURE_RESOURCE_GROUP":
			config.ResourceGroup = val
		case "AZURE_CLOUD_NAME":
			config.CloudName = val
		}
	}

	return config
}

func RenderAzureConfig(config Azure) []byte {
	var writer bytes.Buffer

	// don't care about errors. if this breaks, just turn the lights off on the way out.
	fmt.Fprintf(&writer, "AZURE_SUBSCRIPTION_ID=%s\n", config.SubscriptionID)
	fmt.Fprintf(&writer, "AZURE_TENANT_ID=%s\n", config.TenantID)
	fmt.Fprintf(&writer, "AZURE_CLIENT_ID=%s\n", config.ClientID)
	fmt.Fprintf(&writer, "AZURE_CLIENT_SECRET=%s\n", config.ClientSecret)
	fmt.Fprintf(&writer, "AZURE_RESOURCE_GROUP=%s\n", config.ResourceGroup)
	fmt.Fprintf(&writer, "AZURE_CLOUD_NAME=%s\n", config.CloudName)

	return writer.Bytes()
}
