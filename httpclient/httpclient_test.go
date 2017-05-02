package httpclient_test

import (
	"context"

	"github.com/shurcooL/notificationsapp/httpclient"
	"golang.org/x/oauth2"
)

func ExampleNewNotifications() {
	// HTTP client with authentication.
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "... your access token ..."},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	notificationsClient := httpclient.NewNotifications(httpClient, "http", "localhost:8080")

	// Now you can use any of notificationsClient methods.

	// Output:

	_ = notificationsClient
}
