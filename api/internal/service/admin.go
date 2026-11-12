package service

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"saas-starter/api/internal/store"
)

const (
	adminUsersPageSize = 50
	adminSearchMax     = 254
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type AdminUsersQuery struct {
	Search, Plan, Confirmed string
	Offset                  int
}

type AdminUsersPage struct {
	Users    []store.AdminUser `json:"users"`
	Total    int               `json:"total"`
	Offset   int               `json:"offset"`
	PageSize int               `json:"page_size"`
}

func (s *Service) AdminUsers(ctx context.Context, u store.User, q AdminUsersQuery) (AdminUsersPage, error) {
	if !u.IsAdmin {
		return AdminUsersPage{}, ErrNotFound
	}

	search := strings.TrimSpace(strings.ReplaceAll(strings.ToValidUTF8(q.Search, ""), "\x00", ""))
	if utf8.RuneCountInString(search) > adminSearchMax {
		search = string([]rune(search)[:adminSearchMax])
	}
	f := store.AdminUserFilter{Search: search, Plan: q.Plan}
	if _, ok := plans[q.Plan]; q.Plan != "" && !ok {
		return AdminUsersPage{}, ErrInvalid
	}
	switch q.Confirmed {
	case "":
	case "true", "false":
		yes := q.Confirmed == "true"
		f.Confirmed = &yes
	default:
		return AdminUsersPage{}, ErrInvalid
	}
	offset := max(q.Offset, 0)
	users, total, err := s.store.AdminUsers(ctx, f, adminUsersPageSize, offset)
	if err != nil {
		return AdminUsersPage{}, err
	}
	return AdminUsersPage{Users: users, Total: total, Offset: offset, PageSize: adminUsersPageSize}, nil
}

func (s *Service) AdminUser(ctx context.Context, u store.User, id string) (store.AdminUserDetail, error) {
	if !u.IsAdmin || !uuidRe.MatchString(id) {
		return store.AdminUserDetail{}, ErrNotFound
	}
	return s.store.AdminUserByID(ctx, id)
}
