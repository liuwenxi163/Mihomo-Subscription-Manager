package main

import "time"

type Subscription struct {
	ID         uint        `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	Name       string      `gorm:"not null" json:"name"`
	Token      string      `gorm:"uniqueIndex;not null" json:"token"`
	Links      []ProxyLink `gorm:"constraint:OnDelete:CASCADE" json:"links"`
	RuleGroups []RuleGroup `gorm:"many2many:subscription_rule_groups" json:"rule_groups"`
}

type ProxyLink struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SubscriptionID uint      `gorm:"index;not null" json:"subscription_id"`
	Name           string    `json:"name"`
	Raw            string    `gorm:"type:text;not null" json:"raw"`
	SortOrder      int       `json:"sort_order"`
}

type RuleGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"not null" json:"name"`
	Rules     string    `gorm:"type:text;not null" json:"rules"`
}

type SubscriptionResponse struct {
	ID           uint             `json:"id"`
	Name         string           `json:"name"`
	Token        string           `json:"token"`
	URL          string           `json:"url"`
	Links        []ProxyLinkInput `json:"links"`
	RuleGroupIDs []uint           `json:"rule_group_ids"`
}

type RuleGroupResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Rules string `json:"rules"`
}
