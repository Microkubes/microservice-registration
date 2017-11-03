package config

// SecurityConfig holds the security configuration.
// The subsections for JWT, SAML and OAuth2 are optional. If a
// subsection is ommited, then the appropriate security will not be
// configured and used by the security chain.
type SecurityConfig struct {
	// Disable flags signals whether to disable the security completely.
	Disable bool

	// KeysDir is the loacation of the directory holding the private-public key pairs.
	KeysDir string `json:"keysDir"`

	// JWTConfig holds the JWT configuration. If ommited the JWT security will not be used.
	*JWTConfig `json:"jwt,omitempty"`

	// SAMLConfig holds the SAML configuration. If ommited the SAML security will not be used.
	*SAMLConfig `json:"saml,omitempty"`

	//OAuth2Config holds the OAuth2 configuration. If ommited the OAuth2 security will not be used.
	*OAuth2Config `json:"oauth2,omitempty"`

	// ACL Middleware configuration
	*ACLConfig `json:"acl,omitempty"`
}

// JWTConfig holds the JWT configuration.
type JWTConfig struct {

	// Name is the name of the JWT middleware. Used in error messages.
	Name string

	// Description holds the description for the middleware. Used for documentation purposes.
	Description string

	// TokenURL is the URL of the JWT token provider. Use a full URL here.
	TokenURL string `json:"tokenUrl"`
}

// SAMLConfig holds the SAML configuration.
type SAMLConfig struct {

	// CertFile is the location of the certificate file.
	CertFile string `json:"certFile"`

	// KeyFile is the location of the key file.
	KeyFile string `json:"keyFile"`

	// IdentityProviderURL is the URL of the SAML Identity Provider server. User a full URL here.
	IdentityProviderURL string `json:"identityProviderUrl"`

	// UserServiceURL is the URL of the user microservice. This should be the public url (usually over the Gateway).
	UserServiceURL string `json:"userServiceUrl"`

	// RegistrationServiceURL is the URL of the registration service. This should be the public registration URL (usually over the Gateway).
	RegistrationServiceURL string `json:"registrationServiceUrl"`

	// RootURL is the base URL of the microservice
	RootURL string `json:"rootURL"`
}

// OAuth2Config holds the OAuth2 configuration.
type OAuth2Config struct {

	// TokenURL is the path of the token endpoint. Usually "/oauth2/token".
	TokenURL string `json:"tokenUrl"`

	// AuthorizationURL is the path of the authorize endpoint. Usually "/oauth2/authorize".
	AuthorizationURL string `json:"authorizeUrl"`

	// Description is the description of the middleware. Used for documentation purposes.
	Description string `json:"description"`
}

// ACLConfig holds the ACL middleware configuration.
type ACLConfig struct {

	// Disable signals whether to disable the ACL check.
	Disable bool `json:"disable"`

	// Policies is the list of default policies.
	Policies []ACLPolicy `json:"policies,omitempty"`
}

// ACLPolicy represents an ACL policy
type ACLPolicy struct {
	// The ID of the policy document
	ID string `json:"id" bson:"id"`

	// Description is the human readable description of the document.
	Description string `json:"description" bson:"description"`

	// List of subjects (may be patterns) to which this policy applies.
	Subjects []string `json:"subjects" bson:"subjects"`

	// Effect is the effect of this policy if applied to the requested resource. May be "allow" or "deny".
	Effect string `json:"effect" bson:"effect"`

	// Resources is a list of resources (may be patterns) to which this policy applies.
	Resources []string `json:"resources" bson:"resources"`

	// Actions is a list of actions (may be patterns) to which this policy applies.
	Actions []string `json:"actions" bson:"actions"`

	// Conditions additional ACL policy conditions.
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}
