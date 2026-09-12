package core

import "errors"

// PortalUserIdentity contains the minimum fields needed to authenticate a Trojan user.
type PortalUserIdentity struct {
	UserID         uint
	Username       string
	PasswordDigest string
}

// PortalUserUsage is the minimum account and traffic record used by the personal dashboard.
type PortalUserUsage struct {
	UserID     uint
	Username   string
	Quota      int64
	Download   uint64
	Upload     uint64
	UseDays    uint
	ExpiryDate string
}

// PortalUserIdentityByUsername returns the existing Trojan credential for direct web sign-in.
func (mysql *Mysql) PortalUserIdentityByUsername(username string) (*PortalUserIdentity, error) {
	db := mysql.GetDB()
	if db == nil {
		return nil, errors.New("can't connect mysql")
	}
	defer db.Close()
	identity := PortalUserIdentity{}
	err := db.QueryRow(`SELECT id, username, password
		FROM users
		WHERE BINARY username = ?
		LIMIT 1`, username).Scan(
		&identity.UserID,
		&identity.Username,
		&identity.PasswordDigest,
	)
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

// PortalUserIdentityByID revalidates the user and credential version for every protected request.
func (mysql *Mysql) PortalUserIdentityByID(userID uint) (*PortalUserIdentity, error) {
	db := mysql.GetDB()
	if db == nil {
		return nil, errors.New("can't connect mysql")
	}
	defer db.Close()
	identity := PortalUserIdentity{}
	err := db.QueryRow(`SELECT id, username, password
		FROM users
		WHERE id = ?`, userID).Scan(
		&identity.UserID,
		&identity.Username,
		&identity.PasswordDigest,
	)
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

// PortalUserUsageByID returns the minimum fields required by the personal dashboard.
func (mysql *Mysql) PortalUserUsageByID(userID uint) (*PortalUserUsage, error) {
	db := mysql.GetDB()
	if db == nil {
		return nil, errors.New("can't connect mysql")
	}
	defer db.Close()
	usage := PortalUserUsage{}
	err := db.QueryRow(`SELECT id, username, quota, download, upload, useDays, expiryDate
		FROM users
		WHERE id = ?`, userID).Scan(
		&usage.UserID,
		&usage.Username,
		&usage.Quota,
		&usage.Download,
		&usage.Upload,
		&usage.UseDays,
		&usage.ExpiryDate,
	)
	if err != nil {
		return nil, err
	}
	return &usage, nil
}
