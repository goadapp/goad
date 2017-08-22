package util

func RemoveDuplicates(strs []string) []string {
	strsMap := make(map[string]bool)
	for _, str := range strs {
		strsMap[str] = true
	}
	returnStrs := make([]string, 0)
	for str := range strsMap {
		returnStrs = append(returnStrs, str)
	}
	return returnStrs
}
