package management

type Config struct {
	// Secret is the shared signing secret Bitbucket attaches to deliveries
	// and the service registers on managed hooks.
	Secret string

	// CallbackURL is the public base URL (https scheme://host[/path], no
	// query string or fragment) where Bitbucket webhooks are delivered. The
	// full delivery path is appended by the management service.
	CallbackURL string
}
