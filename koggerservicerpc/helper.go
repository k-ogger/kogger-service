package koggerservicerpc

import "strings"

func ResourceTypeToString(resourceType ResourceType) string {
	fullName := resourceType.String()
	return strings.TrimPrefix(fullName, "RESOURCE_TYPE_")
}

func StringToResourceType(resourceType string) ResourceType {
	fullName := "RESOURCE_TYPE_" + resourceType
	if value, exists := ResourceType_value[fullName]; exists {
		return ResourceType(value)
	}
	return ResourceType_RESOURCE_TYPE_UNKNOWN
}
