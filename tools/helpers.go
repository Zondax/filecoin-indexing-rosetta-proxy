package tools

func IsOpSupported(op string) bool {
	supported, ok := SupportedOperations[op]
	if ok && supported {
		return true
	}

	return false
}

func SetupSupportedOperations(ops []string) {
	for s := range SupportedOperations {
		for _, op := range ops {
			found := false
			if s == op {
				found = true
			}
			SupportedOperations[s] = found
			if found {
				break
			}
		}
	}
}
