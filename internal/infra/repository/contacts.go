package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mikhailjbs/chat-service/internal/domain/contacts"
	"mikhailjbs/chat-service/internal/infra/httpclient"
)

type userRepository struct {
	db         *gorm.DB
	userClient *httpclient.Client
}

func NewContactsRepository(db *gorm.DB, userClient *httpclient.Client) contacts.Repository {
	return &userRepository{db: db, userClient: userClient}
}

func (r *userRepository) CreateContact(contact *contacts.Contact) (*contacts.Contact, error) {
	if err := r.db.Create(contact).Error; err != nil {
		return nil, err
	}
	return contact, nil
}

func (r *userRepository) GetContactByID(id uuid.UUID) (*contacts.Contact, error) {
	var contact contacts.Contact
	if err := r.db.First(&contact, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contacts.ErrContactNotFound
		}
		return nil, err
	}
	return &contact, nil
}

func (r *userRepository) GetContactByUserId(userId uuid.UUID) (*contacts.Contact, error) {
	var contact contacts.Contact
	if err := r.db.First(&contact, "user_id = ?", userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contacts.ErrContactNotFound
		}
		return nil, err
	}
	return &contact, nil
}

func (r *userRepository) UpdateContact(contact *contacts.Contact) (*contacts.Contact, error) {
	if err := r.db.Save(contact).Error; err != nil {
		return nil, err
	}
	return contact, nil
}

func (r *userRepository) DeleteContact(id uuid.UUID) error {
	if err := r.db.Delete(&contacts.Contact{}, "id = ?", id).Error; err != nil {
		return err
	}
	return nil
}

func (r *userRepository) ListContacts(params *contacts.ContactQueryParams) ([]*contacts.Contact, error) {
	var rows []*contacts.Contact
	query := r.db.Model(&contacts.Contact{})
	// Always scope by user when provided to avoid bidirectional duplicates
	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Search != "" {
		query = query.Where("name LIKE ?", "%"+params.Search+"%")
	}
	if params.SortBy != "" {
		order := "asc"
		if params.Order != "" && strings.ToLower(params.Order) == "desc" {
			order = "desc"
		}
		query = query.Order(params.SortBy + " " + order)
	}
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	// Deduplicate by ContactID to avoid accidental duplicates
	uniq := make(map[string]*contacts.Contact, len(rows))
	for _, c := range rows {
		if c == nil {
			continue
		}
		if _, exists := uniq[c.ContactID]; !exists {
			uniq[c.ContactID] = c
		}
	}
	result := make([]*contacts.Contact, 0, len(uniq))
	for _, c := range uniq {
		result = append(result, c)
	}
	return result, nil
}

func (r *userRepository) DeleteContactPair(userID uuid.UUID, contactID uuid.UUID) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("user_id = ? AND contact_id = ?", userID, contactID).Delete(&contacts.Contact{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("user_id = ? AND contact_id = ?", contactID, userID).Delete(&contacts.Contact{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil

}

func (r *userRepository) CreateContactRequest(request *contacts.ContactRequest) (*contacts.ContactRequest, error) {
	if err := r.db.Create(request).Error; err != nil {
		return nil, err
	}
	return request, nil
}

func (r *userRepository) GetContactRequestByID(id uuid.UUID) (*contacts.ContactRequest, error) {
	var request contacts.ContactRequest
	if err := r.db.First(&request, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contacts.ErrContactNotFound
		}
		return nil, err
	}
	return &request, nil
}

func (r *userRepository) UpdateContactRequest(request *contacts.ContactRequest) (*contacts.ContactRequest, error) {
	if err := r.db.Save(request).Error; err != nil {
		return nil, err
	}
	return request, nil
}

func (r *userRepository) DeleteContactRequest(id uuid.UUID) error {
	if err := r.db.Delete(&contacts.ContactRequest{}, "id = ?", id).Error; err != nil {
		return err
	}
	return nil
}

func (r *userRepository) ListContactRequests(params *contacts.ContactRequestQueryParams) ([]*contacts.ContactRequest, error) {
	var requests []*contacts.ContactRequest
	query := r.db.Model(&contacts.ContactRequest{})
	// Apply user-scoped filters when provided
	if params.SenderID != "" && params.ReceiverID != "" {
		query = query.Where("sender_id = ? OR receiver_id = ?", params.SenderID, params.ReceiverID)
	} else if params.SenderID != "" {
		query = query.Where("sender_id = ?", params.SenderID)
	} else if params.ReceiverID != "" {
		query = query.Where("receiver_id = ?", params.ReceiverID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.SortBy != "" {
		order := "asc"
		if params.Order != "" && strings.ToLower(params.Order) == "desc" {
			order = "desc"
		}
		query = query.Order(params.SortBy + " " + order)
	}
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}
	if err := query.Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func (r *userRepository) AcceptContactRequest(request *contacts.ContactRequest, bearerToken string) (*contacts.ContactRequest, error) {
	if r.userClient == nil {
		return nil, errors.New("user auth client is not configured")
	}

	ctx := context.Background()
	senderUser, err := r.FetchUser(ctx, request.SenderID, bearerToken)
	if err != nil {
		return nil, err
	}
	receiverUser, err := r.FetchUser(ctx, request.ReceiverID, bearerToken)
	if err != nil {
		return nil, err
	}

	receiverContact, err := buildContact(request.ReceiverID, senderUser)
	if err != nil {
		return nil, err
	}
	senderContact, err := buildContact(request.SenderID, receiverUser)
	if err != nil {
		return nil, err
	}

	request.Status = "accepted"

	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	if err := tx.Save(request).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(receiverContact).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(senderContact).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return request, nil
}

func (r *userRepository) FetchUser(ctx context.Context, userID string, bearerToken string) (*httpclient.User, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	resp, err := r.userClient.GetUserByID(ctx, userID, bearerToken)
	if err != nil {
		return nil, fmt.Errorf("fetch user %s: %w", userID, err)
	}
	if resp == nil || resp.User == nil {
		return nil, fmt.Errorf("user %s not found", userID)
	}
	return resp.User, nil
}

func buildContact(ownerID string, peer *httpclient.User) (*contacts.Contact, error) {
	if ownerID == "" {
		return nil, errors.New("contact owner id is required")
	}
	if peer == nil {
		return nil, errors.New("peer user payload is required")
	}
	if peer.ID == "" {
		return nil, errors.New("peer user id is missing")
	}
	name := peer.Username
	if peer.FullName != "" {
		name = peer.FullName
	}
	if name == "" {
		name = peer.Email
	}
	if name == "" {
		return nil, fmt.Errorf("peer user %s missing display name", peer.ID)
	}
	if peer.Email == "" {
		return nil, fmt.Errorf("peer user %s missing email", peer.ID)
	}
	username := peer.Username
	if username == "" {
		username = peer.Email
	}
	if username == "" {
		return nil, fmt.Errorf("peer user %s missing username", peer.ID)
	}

	return &contacts.Contact{
		ID:        uuid.NewString(),
		UserID:    ownerID,
		ContactID: peer.ID,
		Name:      name,
		Username:  username,
		Email:     peer.Email,
	}, nil
}
