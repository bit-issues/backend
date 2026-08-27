package oauth

// GenerateState exposes generateState to the external test package. This file is
// only compiled during testing, so it does not expand the production API.
func GenerateState() (string, error) {
	return generateState()
}
