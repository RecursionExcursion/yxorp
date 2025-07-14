package main

type Service struct {
	Name         string
	BaseUrl      string
	PathAlias    string
	ServiceToken string
	Secret       string
	LastUsed     int64
	Enabled      bool
	Secured      bool
	PublicRoutes []string
}
