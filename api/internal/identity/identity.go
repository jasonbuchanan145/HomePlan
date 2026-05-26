package identity

type AuthUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

type AppEntitlement struct {
	CanAccess bool `json:"canAccess"`
	CanUseAI  bool `json:"canUseAI"`
}

type AuthIdentity struct {
	Provider        string
	ProviderSubject string
	Email           string
	EmailVerified   bool
	DisplayName     string
	AvatarURL       string
}

type SignedInSession struct {
	User        AuthUser
	Entitlement AppEntitlement
}
