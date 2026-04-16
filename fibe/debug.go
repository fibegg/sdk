package fibe
import "time"
func (c *Client) HTTPClientTimeout() time.Duration { return c.http.Timeout }
