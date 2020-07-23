package runtime

func Fuzz(data []byte) int {
	err := ParseFeatures(string(data))
	if err != nil {
		return 0
	}
	return 1
}
