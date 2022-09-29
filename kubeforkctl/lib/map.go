package lib

func Merge(m1, m2 map[string]string) map[string]string {
	ans := map[string]string{}

	// If m1 and m2 have the same key, the value after merging is that of m2
	for k, v := range m1 {
		ans[k] = v
	}
	for k, v := range m2 {
		ans[k] = v
	}
	return ans
}
