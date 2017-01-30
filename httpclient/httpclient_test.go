package httpclient_test

import (
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/httpclient"
)

var _ notifications.Service = httpclient.Notifications{}
