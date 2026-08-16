package gapi

import (
	"fmt"

	db "github.com/MumAroi/go-simplebank/db/sqlc"
	"github.com/MumAroi/go-simplebank/pb"
	"github.com/MumAroi/go-simplebank/token"
	"github.com/MumAroi/go-simplebank/util"
)

type Server struct {
	pb.UnimplementedSimpleBankServer
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create token maker: %w", err)
	}

	server := &Server{config: config, store: store, tokenMaker: tokenMaker}

	return server, nil
}
