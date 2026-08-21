package gapi

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type Metadata struct {
	UserAgent string
	ClientIP  string
}

const (
	grpcUserAgentHeader = "grpcgateway-user-agent"
	userAgentHeader     = "user-agent"
	xForwardedForHeader = "x-forwarded-for"
)

func (server *Server) extractMetadata(ctx context.Context) (*Metadata, error) {
	mtdt := &Metadata{}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no metadata in incoming context")
	}

	if userAgent := md.Get(userAgentHeader); len(userAgent) > 0 {
		mtdt.UserAgent = userAgent[0]
	}

	if p, ok := peer.FromContext(ctx); ok {
		mtdt.ClientIP = p.Addr.String()
	}

	if userAgent := md.Get(grpcUserAgentHeader); len(userAgent) > 0 {
		mtdt.UserAgent = userAgent[0]
	}

	if clientIP := md.Get(xForwardedForHeader); len(clientIP) > 0 {
		mtdt.ClientIP = clientIP[0]
	}

	return mtdt, nil
}
