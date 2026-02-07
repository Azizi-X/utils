package sec_ch_ua

import "strings"

var replacer = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func serializeSHString(s string) string {
	return replacer.Replace(s)
}
