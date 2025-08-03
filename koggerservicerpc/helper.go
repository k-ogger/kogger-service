package koggerservicerpc

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ResourceTypeToString(resourceType ResourceType) string {
	fullName := resourceType.String()
	return cases.Title(language.Und).String(strings.TrimPrefix(fullName, "RESOURCE_TYPE_"))
}

func StringToResourceType(resourceType string) ResourceType {
	fullName := "RESOURCE_TYPE_" + strings.ToUpper(resourceType)
	if value, exists := ResourceType_value[fullName]; exists {
		return ResourceType(value)
	}
	return ResourceType_RESOURCE_TYPE_UNKNOWN
}
