package contacts

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"mikhailjbs/chat-service/internal/domain/user"
)

var (
	ErrContactNotFound        = errors.New("contact not found")
	ErrInvalidContact         = errors.New("invalid contact data")
	ErrDuplicateContact       = errors.New("duplicate contact")
	ErrDuplicateRequest       = errors.New("you have already requested to this person")
	ErrContactRequestNotFound = errors.New("contact request not found")
)

type Repository interface {
	CreateContact(contact *Contact) (*Contact, error)
	GetContactByID(id uuid.UUID) (*Contact, error)
	GetContactByUserId(userId uuid.UUID) (*Contact, error)
	UpdateContact(contact *Contact) (*Contact, error)
	DeleteContact(id uuid.UUID) error
	ListContacts(params *ContactQueryParams) ([]*Contact, error)
	DeleteContactPair(userID uuid.UUID, contactID uuid.UUID) error

	CreateContactRequest(request *ContactRequest) (*ContactRequest, error)
	GetContactRequestByID(id uuid.UUID) (*ContactRequest, error)
	UpdateContactRequest(request *ContactRequest) (*ContactRequest, error)
	AcceptContactRequest(request *ContactRequest, bearerToken string) (*ContactRequest, error)
	FetchUser(userID string) (*user.UserSnapshot, error)
	DeleteContactRequest(id uuid.UUID) error
	ListContactRequests(params *ContactRequestQueryParams) ([]*ContactRequest, error)
}

type Service interface {
	GetContactByID(id string) (*Contact, error)
	UpdateContact(contact *UpdateContact, id string) (*Contact, error)
	DeleteContact(userId string, contactId string) error
	ListContacts(params *ContactQueryParams) ([]*Contact, error)

	CreateContactRequest(request *CreateContactRequest, bearerToken string) (*ContactRequest, error)
	GetContactRequestByID(id string) (*ContactRequest, error)
	UpdateContactRequest(request *UpdateContactRequest, id string, bearerToken string) (*ContactRequest, error)
	DeleteContactRequest(id string) error
	ListContactRequests(params *ContactRequestQueryParams) ([]*ContactRequest, error)
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) GetContactByID(id string) (*Contact, error) {
	parsedID := uuid.MustParse(id)
	return s.repo.GetContactByID(parsedID)
}

func (s *service) UpdateContact(contact *UpdateContact, id string) (*Contact, error) {
	parsedID := uuid.MustParse(id)
	existing, err := s.repo.GetContactByID(parsedID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrContactNotFound
	}

	if contact.AssignedName != "" {
		existing.AssignedName = contact.AssignedName
	}
	if contact.Muted != nil {
		existing.Muted = *contact.Muted
	}

	return s.repo.UpdateContact(existing)
}

func (s *service) DeleteContact(userId string, contactId string) error {
	parsedUserID := uuid.MustParse(userId)
	parsedContactID := uuid.MustParse(contactId)

	oppositeContact, err := s.repo.GetContactByUserId(parsedContactID)
	if err != nil {
		return err
	}

	if oppositeContact == nil {
		return ErrContactNotFound
	}

	parsedUserId2 := uuid.MustParse(oppositeContact.UserID)

	return s.repo.DeleteContactPair(parsedUserID, parsedUserId2)
}

func (s *service) ListContacts(params *ContactQueryParams) ([]*Contact, error) {
	return s.repo.ListContacts(params)
}

func (s *service) CreateContactRequest(request *CreateContactRequest, bearerToken string) (*ContactRequest, error) {
	var senderName, receiverName string

	// Fetch sender explicitly for resolving name (if available)
	senderUser, err := s.repo.FetchUser(request.SenderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sender info: %w", err)
	}
	senderName = senderUser.Fullname
	if senderName == "" {
		senderName = senderUser.Username
	}
	if senderName == "" {
		senderName = senderUser.Email
	}

	// Fetch receiver explicitly for resolving name
	receiverUser, err := s.repo.FetchUser(request.ReceiverID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch receiver info: %w", err)
	}
	receiverName = receiverUser.Fullname
	if receiverName == "" {
		receiverName = receiverUser.Username
	}
	if receiverName == "" {
		receiverName = receiverUser.Email
	}

	newRequest := &ContactRequest{
		ID:           uuid.NewString(),
		SenderID:     request.SenderID,
		ReceiverID:   request.ReceiverID,
		SenderName:   senderName,
		ReceiverName: receiverName,
		Message:      request.Message,
		Status:       "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return s.repo.CreateContactRequest(newRequest)
}

func (s *service) GetContactRequestByID(id string) (*ContactRequest, error) {
	parsedID := uuid.MustParse(id)
	return s.repo.GetContactRequestByID(parsedID)
}

func (s *service) UpdateContactRequest(request *UpdateContactRequest, id string, bearerToken string) (*ContactRequest, error) {
	parsedID := uuid.MustParse(id)
	existingRequest, err := s.repo.GetContactRequestByID(parsedID)
	if err != nil {
		return nil, err
	}

	if existingRequest == nil {
		return nil, ErrDuplicateRequest
	}

	if existingRequest.Status == "accepted" || existingRequest.Status == "rejected" {
		return nil, errors.New("contact request has already been processed")
	}

	if request.Status != "" {
		existingRequest.Status = request.Status
	}

	if request.Status == "accepted" {
		return s.repo.AcceptContactRequest(existingRequest, bearerToken)
	}

	existingRequest.Status = request.Status

	return s.repo.UpdateContactRequest(existingRequest)
}

func (s *service) DeleteContactRequest(id string) error {
	parsedID := uuid.MustParse(id)
	return s.repo.DeleteContactRequest(parsedID)
}

func (s *service) ListContactRequests(params *ContactRequestQueryParams) ([]*ContactRequest, error) {
	return s.repo.ListContactRequests(params)
}
