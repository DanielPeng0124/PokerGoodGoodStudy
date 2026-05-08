package auth

import (
	"net/http"

	"github.com/google/uuid"
)

const HeaderUserID = "X-User-ID"
const HeaderUserName = "X-User-Name"

func UserFromRequest(r *http.Request) (id, name string) {
	id = r.Header.Get(HeaderUserID)
	name = r.Header.Get(HeaderUserName)
	// Frontend passes identity via query string for both HTTP and WebSocket calls.
	// Accept these values first so reconnect/refresh preserves the same player identity.
	if id == "" {
		id = r.URL.Query().Get("userId")
	}
	if name == "" {
		name = r.URL.Query().Get("name")
	}
	if id == "" {
		id = uuid.NewString()
	}
	if name == "" {
		name = "Guest-" + id[:8]
	}
	return id, name
}
