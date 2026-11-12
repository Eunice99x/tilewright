package service

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	guestDomain  = "guest.invalid"
	guestIPLimit = 30
	maxNameLen   = 24
)

func isGuest(email string) bool {
	return strings.HasSuffix(email, "@"+guestDomain)
}

func (s *Service) GuestSignup(ctx context.Context, ip, name string) (string, User, error) {
	name, ok := cleanName(name)
	if !ok {
		return "", User{}, ErrInvalid
	}
	if !s.guestsByIP.Hit(ip, s.now()) {
		return "", User{}, ErrTooManyAttempts
	}
	id, err := randomToken()
	if err != nil {
		return "", User{}, err
	}
	u, err := s.store.CreateGuest(ctx, "guest-"+strings.ToLower(id[:16])+"@"+guestDomain, name)
	if err != nil {
		return "", User{}, err
	}
	token, err := s.newSession(ctx, u.ID)
	return token, userView(u), err
}

func cleanName(name string) (string, bool) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || utf8.RuneCountInString(name) > maxNameLen || strings.Contains(name, "@") {
		return "", false
	}
	for _, r := range name {
		if !unicode.IsPrint(r) {
			return "", false
		}
	}
	return name, true
}
