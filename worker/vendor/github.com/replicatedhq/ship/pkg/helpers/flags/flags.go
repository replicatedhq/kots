package flags

import (
	"github.com/spf13/viper"
)

func GetCurrentOrDeprecatedString(v *viper.Viper, currentKey string, deprecatedKey string) string {
	currentKeyValue := v.GetString(currentKey)
	if currentKeyValue == "" {
		return v.GetString(deprecatedKey)
	}
	return currentKeyValue
}
