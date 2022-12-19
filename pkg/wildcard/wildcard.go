package wildcard

func Matches(str, pattern string) bool {
	if pattern == "*" {
		return true
	}
	return runesMatch([]rune(str), []rune(pattern))
}

func runesMatch(str, pattern []rune) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			return runesMatch(str, pattern[1:]) || (len(str) > 0 && runesMatch(str[1:], pattern))
		default:
			if len(str) == 0 || str[0] != pattern[0] {
				return false
			}
		}
		str = str[1:]
		pattern = pattern[1:]
	}

	return len(str) == 0 && len(pattern) == 0
}
