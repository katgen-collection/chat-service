package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type GetContactRequestsUseCase interface {
	Execute(ctx context.Context, params *contacts.ContactRequestQueryParams) ([]*contacts.ContactRequest, error)
}

type getContactRequestsUseCase struct {
	service contacts.Service
}

func NewGetContactRequestsUseCase(service contacts.Service) GetContactRequestsUseCase {
	return &getContactRequestsUseCase{service: service}
}

func (uc *getContactRequestsUseCase) Execute(ctx context.Context, params *contacts.ContactRequestQueryParams) ([]*contacts.ContactRequest, error) {
	return uc.service.ListContactRequests(params)
}
