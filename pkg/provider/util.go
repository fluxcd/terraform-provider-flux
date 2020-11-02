package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func toStringList(ll interface{}) []string {
	if ll == nil {
		return []string{}
	}

	set := ll.(*schema.Set)
	result := []string{}
	for _, l := range set.List() {
		result = append(result, l.(string))
	}

	return result
}
