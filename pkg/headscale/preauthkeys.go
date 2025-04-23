package headscale

import (
	"context"
	"net/http"
	"time"
)

type PreAuthKeyClient struct {
	client HeadscaleClient
}

type PreAuthKey struct {
	User       string    `json:"user"`
	ID         string    `json:"id"`
	Key        string    `json:"key"`
	Reusable   bool      `json:"reusable"`
	Ephemeral  bool      `json:"ephemeral"`
	Used       bool      `json:"used"`
	Expiration time.Time `json:"expiration"`
	CreatedAt  time.Time `json:"createdAt"`
	AclTags    []string  `json:"aclTags"`
}

type ListPreAuthKeysResponse struct {
	PreAuthKeys []PreAuthKey `json:"preAuthKeys"`
}

type CreatePreAuthKeyResponse struct {
	PreAuthKey PreAuthKey `json:"preAuthKey"`
}

type CreatePreAuthKeyRequest struct {
	User       string    `json:"user"`
	Reusable   bool      `json:"reusable"`
	Ephemeral  bool      `json:"ephemeral"`
	Expiration time.Time `json:"expiration"`
	AclTags    []string  `json:"aclTags"`
}

type ExpirePreAuthKeyRequest struct {
	User string `json:"user"`
	Key  string `json:"key"`
}

func (c *PreAuthKeyClient) Create(ctx context.Context, user string, reusable bool, ephemeral bool, expiration time.Time, aclTags []string) (*CreatePreAuthKeyResponse, error) {
	keys := &CreatePreAuthKeyResponse{}
	uri := c.client.buildPath("preauthkey")
	req, err := c.client.buildRequest(ctx, http.MethodPost, uri, request{
		contentType: "application/json",
		body: CreatePreAuthKeyRequest{
			User:       user,
			Reusable:   reusable,
			Ephemeral:  ephemeral,
			Expiration: expiration,
			AclTags:    aclTags,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := c.client.do(ctx, req, keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (c *PreAuthKeyClient) List(ctx context.Context, user string) (*ListPreAuthKeysResponse, error) {
	keys := &ListPreAuthKeysResponse{}
	uri := c.client.buildPath("preauthkey")
	req, err := c.client.buildRequest(ctx, http.MethodGet, uri, request{
		contentType: "application/json",
		params: map[string]string{
			"user": user,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := c.client.do(ctx, req, keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (c *PreAuthKeyClient) Expire(ctx context.Context, user string, key string) error {
	uri := c.client.buildPath("preauthkey", "expire")
	req, err := c.client.buildRequest(ctx, http.MethodPost, uri, request{
		body: ExpirePreAuthKeyRequest{
			User: user,
			Key:  key,
		},
	})
	if err != nil {
		return err
	}
	return c.client.do(ctx, req, nil)
}
