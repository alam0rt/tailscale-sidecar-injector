package headscale

import (
	"context"
	"net/http"
	"time"
)

type UserClient struct {
	client HeadscaleClient
}

type User struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`
	DisplayName   string    `json:"displayName"`
	Email         string    `json:"email"`
	ProviderID    string    `json:"providerId"`
	Provider      string    `json:"provider"`
	ProfilePicURL string    `json:"profilePicUrl"`
}

type UsersResponse struct {
	Users []User `json:"user"`
}

type CreateUserRequest struct {
	Name string `json:"name"`
}

func (u *UserClient) Create(ctx context.Context, name string) (*User, error) {
	user := &User{}

	uri := u.client.buildPath("user")
	req, err := u.client.buildRequest(ctx, http.MethodPost, uri, request{
		body: CreateUserRequest{
			Name: name,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := u.client.do(ctx, req, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *UserClient) List(ctx context.Context) (*UsersResponse, error) {
	users := &UsersResponse{}

	uri := u.client.buildPath("user")
	req, err := u.client.buildRequest(ctx, http.MethodPost, uri, request{})
	if err != nil {
		return nil, err
	}

	if err := u.client.do(ctx, req, users); err != nil {
		return nil, err
	}
	return users, nil
}
