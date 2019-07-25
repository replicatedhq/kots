package upstream

import "net/url"

func isURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	return true
}
