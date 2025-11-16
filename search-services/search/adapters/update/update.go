package update

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	log    *slog.Logger
	Client updatepb.UpdateClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		log:    log,
		Client: updatepb.NewUpdateClient(conn),
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Client.Ping(ctx, nil)
	return err
}

func (c *Client) Update(ctx context.Context) error {
	_, err := c.Client.Update(ctx, nil)
	return err
}
