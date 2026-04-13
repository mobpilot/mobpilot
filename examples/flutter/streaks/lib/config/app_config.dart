/// Configuration for connecting to a Mobpilot backend instance.
class AppConfig {
  /// Base URL for the Mobpilot Identity API (via Traefik gateway).
  final String identityBaseUrl;

  /// Ory Kratos public URL for self-service auth flows.
  final String kratosBaseUrl;

  /// Ory Hydra public URL for OAuth2 token exchange.
  final String hydraBaseUrl;

  /// OAuth2 client ID registered in Hydra.
  final String oauthClientId;

  /// OAuth2 redirect URI for this app.
  final String oauthRedirectUri;

  /// If non-null, skip auth and use this token for all API calls.
  final String? devToken;

  /// If true, enable test mode features (e.g. reset state).
  final bool testMode;

  const AppConfig({
    this.identityBaseUrl = 'http://localhost/v1/identity',
    this.kratosBaseUrl = 'http://localhost:4433',
    this.hydraBaseUrl = 'http://localhost:4444',
    this.oauthClientId = 'mobpilot-dev-app',
    this.oauthRedirectUri = 'http://localhost:3000/callback',
    this.devToken,
    this.testMode = false,
  });

  /// Parse CLI arguments into an AppConfig.
  factory AppConfig.fromArgs(List<String> args) {
    String identityBaseUrl = 'http://localhost/v1/identity';
    String kratosBaseUrl = 'http://localhost:4433';
    String hydraBaseUrl = 'http://localhost:4444';
    String oauthClientId = 'mobpilot-dev-app';
    String oauthRedirectUri = 'http://localhost:3000/callback';
    String? devToken;
    bool testMode = false;

    for (int i = 0; i < args.length; i++) {
      switch (args[i]) {
        case '--base-url':
          if (i + 1 < args.length) identityBaseUrl = args[++i];
        case '--kratos-url':
          if (i + 1 < args.length) kratosBaseUrl = args[++i];
        case '--hydra-url':
          if (i + 1 < args.length) hydraBaseUrl = args[++i];
        case '--client-id':
          if (i + 1 < args.length) oauthClientId = args[++i];
        case '--redirect-uri':
          if (i + 1 < args.length) oauthRedirectUri = args[++i];
        case '--dev-token':
          if (i + 1 < args.length) devToken = args[++i];
        case '--test-mode':
          testMode = true;
      }
    }

    return AppConfig(
      identityBaseUrl: identityBaseUrl,
      kratosBaseUrl: kratosBaseUrl,
      hydraBaseUrl: hydraBaseUrl,
      oauthClientId: oauthClientId,
      oauthRedirectUri: oauthRedirectUri,
      devToken: devToken,
      testMode: testMode,
    );
  }
}
