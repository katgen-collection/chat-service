package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type GetContactsUseCase interface {
	Execute(ctx context.Context, params *contacts.ContactQueryParams) ([]*contacts.Contact, error)
}

type getContactsUseCase struct {
	service contacts.Service
}

func NewGetContactsUseCase(service contacts.Service) GetContactsUseCase {
	return &getContactsUseCase{service: service}
}

func (uc *getContactsUseCase) Execute(ctx context.Context, params *contacts.ContactQueryParams) ([]*contacts.Contact, error) {
	return uc.service.ListContacts(params)
}
