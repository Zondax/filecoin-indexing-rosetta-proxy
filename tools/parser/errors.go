package parser

import "errors"

var (
	errUnknownMethod         = errors.New("not known method")
	errNotActorCreationEvent = errors.New("not an actor creation event")
	errBlockHash             = errors.New("unable to get block hash")
	errNotValidActor         = errors.New("not a valid actor")
)
