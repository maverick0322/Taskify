package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidCreditCardID         = errors.New("domain: credit card ID cannot be empty")
	ErrInvalidCreditCardUserID     = errors.New("domain: credit card user ID cannot be empty")
	ErrInvalidCreditCardName       = errors.New("domain: credit card name cannot be empty")
	ErrInvalidCreditCardBank       = errors.New("domain: credit card bank cannot be empty")
	ErrInvalidCreditCardLast4      = errors.New("domain: credit card last4 must contain exactly four digits")
	ErrInvalidCreditCardCutoffDay  = errors.New("domain: credit card cutoff day must be between 1 and 31")
	ErrInvalidCreditCardPaymentDay = errors.New("domain: credit card payment day must be between 1 and 31")
	ErrInvalidCreditCardLimit      = errors.New("domain: credit card limit must be greater than zero")
	ErrInvalidCreditCardColor      = errors.New("domain: credit card color cannot be empty")
	ErrInvalidCreditCardCreatedAt  = errors.New("domain: credit card created at cannot be zero")
	ErrInvalidCreditCardUpdatedAt  = errors.New("domain: credit card updated at cannot be zero")
)

type CreditCard struct {
	id         string
	userID     string
	name       string
	bank       string
	last4      string
	cutoffDay  int
	paymentDay int
	limitCents int64
	color      string
	createdAt  time.Time
	updatedAt  time.Time
}

func NewCreditCard(id, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) (*CreditCard, error) {
	fields, err := validateCreditCardFields(id, userID, name, bank, last4, cutoffDay, paymentDay, limitCents, color)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	return &CreditCard{
		id:         fields.id,
		userID:     fields.userID,
		name:       fields.name,
		bank:       fields.bank,
		last4:      fields.last4,
		cutoffDay:  cutoffDay,
		paymentDay: paymentDay,
		limitCents: limitCents,
		color:      fields.color,
		createdAt:  currentTime,
		updatedAt:  currentTime,
	}, nil
}

func RehydrateCreditCard(id, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string, createdAt, updatedAt time.Time) (*CreditCard, error) {
	fields, err := validateCreditCardFields(id, userID, name, bank, last4, cutoffDay, paymentDay, limitCents, color)
	if err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidCreditCardCreatedAt
	}
	if updatedAt.IsZero() {
		return nil, ErrInvalidCreditCardUpdatedAt
	}

	return &CreditCard{
		id:         fields.id,
		userID:     fields.userID,
		name:       fields.name,
		bank:       fields.bank,
		last4:      fields.last4,
		cutoffDay:  cutoffDay,
		paymentDay: paymentDay,
		limitCents: limitCents,
		color:      fields.color,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}, nil
}

func (card *CreditCard) Update(name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) error {
	fields, err := validateCreditCardFields(card.id, card.userID, name, bank, last4, cutoffDay, paymentDay, limitCents, color)
	if err != nil {
		return err
	}

	card.name = fields.name
	card.bank = fields.bank
	card.last4 = fields.last4
	card.cutoffDay = cutoffDay
	card.paymentDay = paymentDay
	card.limitCents = limitCents
	card.color = fields.color
	card.touch()
	return nil
}

func (card *CreditCard) ID() string {
	return card.id
}

func (card *CreditCard) UserID() string {
	return card.userID
}

func (card *CreditCard) Name() string {
	return card.name
}

func (card *CreditCard) Bank() string {
	return card.bank
}

func (card *CreditCard) Last4() string {
	return card.last4
}

func (card *CreditCard) CutoffDay() int {
	return card.cutoffDay
}

func (card *CreditCard) PaymentDay() int {
	return card.paymentDay
}

func (card *CreditCard) LimitCents() int64 {
	return card.limitCents
}

func (card *CreditCard) Color() string {
	return card.color
}

func (card *CreditCard) CreatedAt() time.Time {
	return card.createdAt
}

func (card *CreditCard) UpdatedAt() time.Time {
	return card.updatedAt
}

func (card *CreditCard) touch() {
	card.updatedAt = time.Now()
}

type validatedCreditCardFields struct {
	id     string
	userID string
	name   string
	bank   string
	last4  string
	color  string
}

func validateCreditCardFields(id, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) (validatedCreditCardFields, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return validatedCreditCardFields{}, ErrInvalidCreditCardID
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return validatedCreditCardFields{}, ErrInvalidCreditCardUserID
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return validatedCreditCardFields{}, ErrInvalidCreditCardName
	}

	trimmedBank := strings.TrimSpace(bank)
	if trimmedBank == "" {
		return validatedCreditCardFields{}, ErrInvalidCreditCardBank
	}

	trimmedLast4 := strings.TrimSpace(last4)
	if !isFourDigitCreditCardSuffix(trimmedLast4) {
		return validatedCreditCardFields{}, ErrInvalidCreditCardLast4
	}

	if cutoffDay < 1 || cutoffDay > 31 {
		return validatedCreditCardFields{}, ErrInvalidCreditCardCutoffDay
	}

	if paymentDay < 1 || paymentDay > 31 {
		return validatedCreditCardFields{}, ErrInvalidCreditCardPaymentDay
	}

	if limitCents <= 0 {
		return validatedCreditCardFields{}, ErrInvalidCreditCardLimit
	}

	trimmedColor := strings.TrimSpace(color)
	if trimmedColor == "" {
		return validatedCreditCardFields{}, ErrInvalidCreditCardColor
	}

	return validatedCreditCardFields{
		id:     trimmedID,
		userID: trimmedUserID,
		name:   trimmedName,
		bank:   trimmedBank,
		last4:  trimmedLast4,
		color:  trimmedColor,
	}, nil
}

func isFourDigitCreditCardSuffix(last4 string) bool {
	if len(last4) != 4 {
		return false
	}

	for _, digit := range last4 {
		if !unicode.IsDigit(digit) {
			return false
		}
	}

	return true
}
