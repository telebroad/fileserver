package users

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"net/netip"
	"strings"
	"sync"
)

type User struct {
	Username string
	Password string
	IPs      map[string]*netip.Prefix
}

func UniqSlice[T comparable](s []T) []T {
	m := make(map[T]struct{})
	for _, v := range s {
		m[v] = struct{}{}
	}
	var result []T
	for k := range m {
		result = append(result, k)
	}
	return result
}

// FindIP finds an IP in the prefixes in the user
func (u *User) FindIP(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for k, v := range u.IPs {
		if k == "*" {
			return true
		}
		if v.Contains(addr) {
			return true
		}
	}
	return false
}

// AddIP adds an IP prefix to the user
// if the ip is without the prefix, it will add /32
func (u *User) AddIP(ip string) (err error) {
	if ip == "*" {
		u.IPs["*"] = nil
		return
	}
	var prefix netip.Prefix
	if !strings.Contains(ip, "/") {

		addr, err := netip.ParseAddr(ip)
		if err != nil {
			return fmt.Errorf("error parsing IP: %w", err)
		}
		if addr.Is4() {
			prefix, _ = addr.Prefix(32)
		} else {
			prefix, _ = addr.Prefix(128)
		}
	} else {
		prefix, err = netip.ParsePrefix(ip)
		if err != nil {
			return fmt.Errorf("error parsing IP: %w", err)
		}
	}

	u.IPs[ip] = &prefix
	return nil
}

// RemoveIP removes an IP prefix from the user
// if the ip is without the prefix, it will add /32
func (u *User) RemoveIP(ip string) {

	if ip == "*" {
		delete(u.IPs, "*")
		return
	}

	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}

	delete(u.IPs, ip)
}

var localUserMaxID int64 = 0

type LocalUsers struct {
	logger *slog.Logger
	users  map[string]*User
	wg     sync.RWMutex
}

func (u *LocalUsers) Logger() *slog.Logger {
	if u.logger == nil {
		u.logger = slog.Default()
	}
	return u.logger.With("module", "users")
}

// List returns all users
func (u *LocalUsers) List() (map[string]*User, error) {
	u.wg.RLock()
	defer u.wg.RUnlock()
	return u.users, nil
}

// Get returns a user by username, if the user is not found it returns an error
func (u *LocalUsers) Get(username string) (*User, error) {
	u.wg.RLock()
	defer u.wg.RUnlock()
	user, ok := u.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// FindUser returns a user by username and password, if the user is not found it returns an error
func (u *LocalUsers) FindUser(ctx context.Context, username, password, ipaddr string) (any, error) {
	userInfo, err := u.Get(username)
	if err != nil {
		u.Logger().Debug("user not found", "user", username)
		return nil, err
	}
	if userInfo.Password != password {
		u.Logger().Debug("password is incorrect", "user", username)
		return nil, fmt.Errorf("password is incorrect")
	}
	if strings.Contains(ipaddr, ":") {
		ipaddr = strings.Split(ipaddr, ":")[0]
	}
	if !userInfo.FindIP(ipaddr) {
		u.Logger().Debug("ip origin", ipaddr, "is not allowed", "user", username)
		return nil, fmt.Errorf("ip origin %s is not allowed", ipaddr)
	}
	return userInfo, nil
}

func (u *LocalUsers) VerifyUser(r *http.Request) (any, error) {

	user, pass, ok := r.BasicAuth()

	if !ok {
		// Inform the client about the required authentication scheme and provide an example

		exampleUser := "exampleUser"
		examplePass := "examplePass"
		exampleCredentials := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", exampleUser, examplePass)))

		errorMessage := fmt.Errorf("access denied. This resource requires Basic authentication. For example, "+
			"set the Authorization header as: Authorization: Basic %s or http://%s:%s@example.com",
			exampleCredentials, exampleUser, examplePass)
		u.Logger().Debug("access denied", "error", errorMessage, "remoteAddr", r.RemoteAddr, "user-agent", r.UserAgent())
		return nil, errorMessage
	}
	return u.FindUser(r.Context(), user, pass, r.RemoteAddr)

}

// Add adds a new user
func (u *LocalUsers) Add(user, pass string) *User {
	u.wg.Lock()
	defer u.wg.Unlock()

	newUser := &User{
		Username: user,
		Password: pass,
		IPs:      make(map[string]*netip.Prefix),
	}

	u.users[newUser.Username] = newUser
	return newUser
}

// Remove removes a user
func (u *LocalUsers) Remove(user string) *User {
	u.wg.Lock()
	defer u.wg.Unlock()
	oldUser := u.users[user]
	delete(u.users, user)
	return oldUser
}

// NewLocalUsers creates a new LocalUsers
func NewLocalUsers(logger *slog.Logger) *LocalUsers {
	return &LocalUsers{
		logger: logger,
		users:  make(map[string]*User),
		wg:     sync.RWMutex{},
	}
}
