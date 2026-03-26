package server

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health checks (no prefix)
	s.router.GET("/api/health", s.healthHandler.Health)
	s.router.GET("/api/health/ready", s.healthHandler.Ready)

	// API v1
	v1 := s.router.Group("/api/v1")

	// Public auth routes
	auth := v1.Group("/auth")
	{
		auth.GET("/challenge", s.authHandler.GetChallenge)
		auth.POST("/verify", s.authHandler.VerifyWalletSignature)
	}

	// Account routes
	accounts := v1.Group("/accounts")
	{
		accounts.GET("/eoa/:eoa_address", s.accountHandler.GetAccountByEOA)
		accounts.GET("/:account_id", s.accountHandler.GetAccount)
		accounts.GET("/:account_id/identities", s.accountHandler.GetAccountIdentities)
		accounts.GET("/:account_id/sessions", s.accountHandler.GetAccountSessions)
	}

	// Identity management
	identities := v1.Group("/identities")
	{
		identities.DELETE("/:identity_id", s.accountHandler.UnlinkIdentity)
	}

	// OAuth2 webhooks (Hydra integration)
	oauth2 := v1.Group("/oauth2")
	{
		oauth2.GET("/login", s.oauth2Handler.HandleLogin)
		oauth2.GET("/consent", s.oauth2Handler.HandleConsent)
		oauth2.GET("/logout", s.oauth2Handler.HandleLogout)
		oauth2.POST("/registration-webhook", s.oauth2Handler.HandleRegistrationWebhook)
	}

	// Authorization routes (SpiceDB)
	if s.authzHandler != nil {
		authz := v1.Group("/authz")
		{
			authz.GET("/health", s.authzHandler.HealthCheck)
			authz.POST("/check", s.authzHandler.CheckPermission)
		}

		// Internal SpiceDB management
		spicedb := s.router.Group("/internal/spicedb")
		{
			spicedb.POST("/relationships", s.authzHandler.CreateRelationship)
			spicedb.DELETE("/relationships", s.authzHandler.DeleteRelationship)
		}
	}

	// App Client Administration (Should be protected by SpiceDB checks in real life, e.g. require 'admin' role)
	// For now, these are accessible without authz wrapper simply to allow testing.
	adminClients := v1.Group("/admin/clients")
	{
		adminClients.POST("", s.clientHandler.CreateClient)
		adminClients.GET("", s.clientHandler.ListClients)
		adminClients.GET("/:id", s.clientHandler.GetClient)
		adminClients.PUT("/:id", s.clientHandler.UpdateClient)
		adminClients.DELETE("/:id", s.clientHandler.DeleteClient)
	}

	// SIWE wallet authentication
	siwe := v1.Group("/siwe")
	{
		siwe.GET("/nonce", s.siweHandler.GetNonce)
		siwe.POST("/verify", s.siweHandler.Verify)
		siwe.POST("/authenticate", s.siweHandler.Authenticate)
	}

	// Message Template Administration
	adminTemplates := v1.Group("/admin/message-templates")
	{
		adminTemplates.POST("", s.templateHandler.CreateTemplate)
		adminTemplates.GET("", s.templateHandler.ListTemplates)
		adminTemplates.GET("/:id", s.templateHandler.GetTemplate)
		adminTemplates.PUT("/:id", s.templateHandler.UpdateTemplate)
		adminTemplates.DELETE("/:id", s.templateHandler.DeleteTemplate)
	}

	// Web2.5 Smart Wallet Routes
	wallet := v1.Group("/wallet")
	{
		wallet.GET("/address/:account_id", s.smartWalletHandler.GetAddress)
		wallet.POST("/execute", s.smartWalletHandler.ExecuteTransaction)
	}
}
