package plugins

import "net/url"

// SetURL sets an URL for the client.
func (c *Client) SetURL(u *url.URL) {
	c.baseURL = u
}
