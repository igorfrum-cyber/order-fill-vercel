package domain

type Credential struct {
	UserID             string
	Secret             string
	Enabled            bool
	RecoveryCodeHashes []string
}

func (c Credential) IsEnabled() bool {
	return c.Enabled && c.Secret != ""
}
