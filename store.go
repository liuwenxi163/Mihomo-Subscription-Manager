package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

type SubscriptionInput struct {
	Name         string           `json:"name"`
	Links        []ProxyLinkInput `json:"links"`
	RuleGroupIDs []uint           `json:"rule_group_ids"`
}

type ProxyLinkInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RuleGroupInput struct {
	Name  string `json:"name"`
	Rules string `json:"rules"`
}

type ExportData struct {
	Subscriptions []ExportSubscription `json:"subscriptions"`
	RuleGroups    []RuleGroupResponse  `json:"rule_groups"`
}

type ExportSubscription struct {
	Name           string           `json:"name"`
	Token          string           `json:"token"`
	Links          []ProxyLinkInput `json:"links"`
	RuleGroupNames []string         `json:"rule_group_names"`
}

func OpenStore(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Subscription{}, &ProxyLink{}, &RuleGroup{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) CreateSubscription(input SubscriptionInput) (*Subscription, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("name is required")
	}
	sub := &Subscription{Name: strings.TrimSpace(input.Name), Token: newToken(), Links: linkModels(input.Links)}
	if err := s.db.Create(sub).Error; err != nil {
		return nil, err
	}
	if err := s.setSubscriptionRuleGroups(sub, input.RuleGroupIDs); err != nil {
		return nil, err
	}
	return s.GetSubscriptionByID(sub.ID)
}

func (s *Store) UpdateSubscription(id uint, input SubscriptionInput) (*Subscription, error) {
	var sub Subscription
	if err := s.db.First(&sub, id).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("name is required")
	}
	return &sub, s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&sub).Update("name", strings.TrimSpace(input.Name)).Error; err != nil {
			return err
		}
		if err := tx.Where("subscription_id = ?", sub.ID).Delete(&ProxyLink{}).Error; err != nil {
			return err
		}
		for _, link := range linkModels(input.Links) {
			link.SubscriptionID = sub.ID
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		var groups []RuleGroup
		if len(input.RuleGroupIDs) > 0 {
			if err := tx.Find(&groups, input.RuleGroupIDs).Error; err != nil {
				return err
			}
		}
		return tx.Model(&sub).Association("RuleGroups").Replace(groups)
	})
}

func (s *Store) DeleteSubscription(id uint) error {
	return s.db.Delete(&Subscription{}, id).Error
}

func (s *Store) GetSubscriptionByID(id uint) (*Subscription, error) {
	var sub Subscription
	err := s.db.Preload("Links", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order asc") }).Preload("RuleGroups").First(&sub, id).Error
	return &sub, err
}

func (s *Store) GetSubscriptionByToken(token string) (*Subscription, error) {
	var sub Subscription
	err := s.db.Preload("Links", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order asc") }).Preload("RuleGroups").Where("token = ?", token).First(&sub).Error
	return &sub, err
}

func (s *Store) ListSubscriptions() ([]Subscription, error) {
	var subs []Subscription
	err := s.db.Preload("Links", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order asc") }).Preload("RuleGroups").Order("id desc").Find(&subs).Error
	return subs, err
}

func (s *Store) CreateRuleGroup(input RuleGroupInput) (*RuleGroup, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("name is required")
	}
	group := &RuleGroup{Name: strings.TrimSpace(input.Name), Rules: strings.TrimSpace(input.Rules)}
	return group, s.db.Create(group).Error
}

func (s *Store) UpdateRuleGroup(id uint, input RuleGroupInput) (*RuleGroup, error) {
	var group RuleGroup
	if err := s.db.First(&group, id).Error; err != nil {
		return nil, err
	}
	group.Name = strings.TrimSpace(input.Name)
	group.Rules = strings.TrimSpace(input.Rules)
	return &group, s.db.Save(&group).Error
}

func (s *Store) DeleteRuleGroup(id uint) error {
	return s.db.Delete(&RuleGroup{}, id).Error
}

func (s *Store) ListRuleGroups() ([]RuleGroup, error) {
	var groups []RuleGroup
	err := s.db.Order("id desc").Find(&groups).Error
	return groups, err
}

func (s *Store) Export() (*ExportData, error) {
	subs, err := s.ListSubscriptions()
	if err != nil {
		return nil, err
	}
	groups, err := s.ListRuleGroups()
	if err != nil {
		return nil, err
	}
	data := &ExportData{}
	for _, group := range groups {
		data.RuleGroups = append(data.RuleGroups, RuleGroupResponse{ID: group.ID, Name: group.Name, Rules: group.Rules})
	}
	for _, sub := range subs {
		item := ExportSubscription{Name: sub.Name, Token: sub.Token}
		for _, link := range sub.Links {
			item.Links = append(item.Links, ProxyLinkInput{Name: link.Name, URL: link.Raw})
		}
		for _, group := range sub.RuleGroups {
			item.RuleGroupNames = append(item.RuleGroupNames, group.Name)
		}
		data.Subscriptions = append(data.Subscriptions, item)
	}
	return data, nil
}

func (s *Store) Import(data ExportData) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM subscription_rule_groups").Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ProxyLink{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Subscription{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RuleGroup{}).Error; err != nil {
			return err
		}
		groupsByName := map[string]RuleGroup{}
		for _, group := range data.RuleGroups {
			model := RuleGroup{Name: group.Name, Rules: group.Rules}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
			groupsByName[group.Name] = model
		}
		for _, exported := range data.Subscriptions {
			sub := Subscription{Name: exported.Name, Token: valueOr(exported.Token, newToken()), Links: linkModels(exported.Links)}
			if err := tx.Create(&sub).Error; err != nil {
				return err
			}
			var selected []RuleGroup
			for _, name := range exported.RuleGroupNames {
				if group, ok := groupsByName[name]; ok {
					selected = append(selected, group)
				}
			}
			if err := tx.Model(&sub).Association("RuleGroups").Replace(selected); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) setSubscriptionRuleGroups(sub *Subscription, ids []uint) error {
	var groups []RuleGroup
	if len(ids) > 0 {
		if err := s.db.Find(&groups, ids).Error; err != nil {
			return err
		}
	}
	return s.db.Model(sub).Association("RuleGroups").Replace(groups)
}

func linkModels(rawLinks []ProxyLinkInput) []ProxyLink {
	links := make([]ProxyLink, 0, len(rawLinks))
	for i, input := range rawLinks {
		raw := strings.TrimSpace(input.URL)
		if raw == "" {
			continue
		}
		links = append(links, ProxyLink{Name: strings.TrimSpace(input.Name), Raw: raw, SortOrder: i})
	}
	return links
}

func newToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte("fallback-token"))
	}
	return hex.EncodeToString(b[:])
}
