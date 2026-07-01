package model

import "slices"

type UserRole string

const (
	RoleUser  UserRole = "ROLE_USER"
	RoleAdmin UserRole = "ROLE_ADMIN"
)

type UserRoles []UserRole

func (r UserRoles) HasRole(role UserRole) bool {
	return slices.Contains(r, role)
}

func (r UserRoles) IsAdmin() bool {
	return r.HasRole(RoleAdmin)
}

func (r UserRoles) ToStrings() []string {
	out := make([]string, len(r))
	for i, v := range r {
		out[i] = string(v)
	}
	return out
}

func UserRolesFromStrings(ss []string) UserRoles {
	out := make(UserRoles, len(ss))
	for i, s := range ss {
		out[i] = UserRole(s)
	}
	return out
}
