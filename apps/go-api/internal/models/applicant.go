package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Applicant struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PositionID uuid.UUID `gorm:"type:uuid;index;not null" json:"positionId"`
	Position   Position  `gorm:"foreignKey:PositionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Name       string    `gorm:"not null" json:"name"`
	Email      string    `gorm:"uniqueIndex;not null" json:"email"`
	Phone      string    `gorm:"uniqueIndex;not null" json:"phone"`
	Resume     string    `json:"resume"`

	CallSummary string     `json:"callSummary,omitempty"`
	CallEndedAt *time.Time `json:"callEndedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (a *Applicant) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
