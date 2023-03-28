package collections

// ListIntersection returns all the items in both list1 and list2. Note that this will dedup the items so that the
// output is more predictable. Otherwise, the end list depends on which list was used as the base.
func ListIntersection(list1 []string, list2 []string) []string {
	out := []string{}

	// Only need to iterate list1, because we want items in both lists, not union.
	for _, item := range list1 {
		if ListContains(list2, item) && !ListContains(out, item) {
			out = append(out, item)
		}
	}

	return out
}

// ListSubtract removes all the items in list2 from list1.
func ListSubtract(list1 []string, list2 []string) []string {
	out := []string{}

	for _, item := range list1 {
		if !ListContains(list2, item) {
			out = append(out, item)
		}
	}

	return out
}

// ListContains returns true if the given list of strings (haystack) contains the given string (needle).
func ListContains(haystack []string, needle string) bool {
	for _, str := range haystack {
		if needle == str {
			return true
		}
	}

	return false
}
