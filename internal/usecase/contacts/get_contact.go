package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type GetContactUseCase interface {
	Execute(ctx context.Context, id string) (*contacts.Contact, error)
}

type getContactUseCase struct {
	service contacts.Service
}

func NewGetContactUseCase(service contacts.Service) GetContactUseCase {
	return &getContactUseCase{service: service}
}

func (uc *getContactUseCase) Execute(ctx context.Context, id string) (*contacts.Contact, error) {
	return uc.service.GetContactByID(id)
}
