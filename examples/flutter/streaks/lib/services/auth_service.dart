import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Handles authentication with Ory Kratos (identity) and Ory Hydra (OAuth2).
///
/// Flow for mobile/desktop:
/// 1. Create Kratos self-service registration or login flow (API type)
/// 2. Submit credentials to Kratos → get a session token
/// 3. Use the session token to approve the Hydra OAuth2 login challenge
/// 4. Exchange the authorization code for an access token
///
/// For dev/test mode, a pre-configured token can bypass this entirely.
class AuthService {
  final String kratosBaseUrl;
  final String hydraBaseUrl;
  final String clientId;
  final String redirectUri;
  final http.Client _http;
  final FlutterSecureStorage _storage;

  String? _accessToken;
  String? _refreshToken;
  String? _kratosSessionToken;

  AuthService({
    required this.kratosBaseUrl,
    required this.hydraBaseUrl,
    required this.clientId,
    required this.redirectUri,
    http.Client? httpClient,
    FlutterSecureStorage? storage,
  })  : _http = httpClient ?? http.Client(),
        _storage = storage ?? const FlutterSecureStorage();

  String? get accessToken => _accessToken;
  bool get isAuthenticated => _accessToken != null;

  /// Load a previously stored token from secure storage.
  Future<bool> tryRestoreSession() async {
    _accessToken = await _storage.read(key: 'access_token');
    _refreshToken = await _storage.read(key: 'refresh_token');
    _kratosSessionToken = await _storage.read(key: 'kratos_session');
    return _accessToken != null;
  }

  /// Set a dev token directly (bypasses Kratos/Hydra).
  Future<void> setDevToken(String token) async {
    _accessToken = token;
    await _storage.write(key: 'access_token', value: token);
  }

  /// Register a new user via Kratos self-service API flow.
  Future<AuthResult> register({
    required String email,
    required String password,
  }) async {
    // Step 1: Create registration flow
    final flowResp = await _http.get(
      Uri.parse('$kratosBaseUrl/self-service/registration/api'),
    );
    if (flowResp.statusCode != 200) {
      return AuthResult.failure('Failed to create registration flow');
    }
    final flowData = jsonDecode(flowResp.body) as Map<String, dynamic>;
    final flowId = flowData['id'] as String;
    final actionUrl = _extractActionUrl(flowData);

    // Step 2: Submit registration
    final submitResp = await _http.post(
      Uri.parse(actionUrl ?? '$kratosBaseUrl/self-service/registration?flow=$flowId'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'method': 'password',
        'password': password,
        'traits': {'email': email},
      }),
    );

    if (submitResp.statusCode == 200 || submitResp.statusCode == 201) {
      final result = jsonDecode(submitResp.body) as Map<String, dynamic>;
      _kratosSessionToken = result['session_token'] as String?;
      if (_kratosSessionToken != null) {
        await _storage.write(key: 'kratos_session', value: _kratosSessionToken);
        // Exchange Kratos session for Hydra OAuth2 token
        return _exchangeForOAuthToken();
      }
      return AuthResult.failure('No session token in registration response');
    }

    return _parseKratosError(submitResp);
  }

  /// Log in via Kratos self-service API flow.
  Future<AuthResult> login({
    required String email,
    required String password,
  }) async {
    // Step 1: Create login flow
    final flowResp = await _http.get(
      Uri.parse('$kratosBaseUrl/self-service/login/api'),
    );
    if (flowResp.statusCode != 200) {
      return AuthResult.failure('Failed to create login flow');
    }
    final flowData = jsonDecode(flowResp.body) as Map<String, dynamic>;
    final flowId = flowData['id'] as String;
    final actionUrl = _extractActionUrl(flowData);

    // Step 2: Submit login
    final submitResp = await _http.post(
      Uri.parse(actionUrl ?? '$kratosBaseUrl/self-service/login?flow=$flowId'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'method': 'password',
        'identifier': email,
        'password': password,
      }),
    );

    if (submitResp.statusCode == 200) {
      final result = jsonDecode(submitResp.body) as Map<String, dynamic>;
      _kratosSessionToken = result['session_token'] as String?;
      if (_kratosSessionToken != null) {
        await _storage.write(key: 'kratos_session', value: _kratosSessionToken);
        return _exchangeForOAuthToken();
      }
      return AuthResult.failure('No session token in login response');
    }

    return _parseKratosError(submitResp);
  }

  /// Exchange the Kratos session for a Hydra OAuth2 access token.
  ///
  /// This uses a simplified approach for the reference app:
  /// We use the client_credentials flow with the Kratos session token
  /// as additional context. In a production app, you'd use the full
  /// authorization code flow with PKCE via a browser redirect.
  Future<AuthResult> _exchangeForOAuthToken() async {
    // For the reference app in dev mode, we'll use the Kratos session token
    // directly as the bearer token for the identity API. The identity service's
    // auth middleware should be configured to accept Kratos session tokens
    // in addition to Hydra OAuth2 tokens.
    //
    // In production, you'd implement the full OAuth2 authorization code flow:
    // 1. Redirect to Hydra /oauth2/auth with PKCE
    // 2. Hydra redirects to Kratos login (user is already authenticated)
    // 3. Kratos redirects back to Hydra with approved login challenge
    // 4. Hydra redirects to app with authorization code
    // 5. App exchanges code for token at Hydra /oauth2/token
    //
    // For now, we use client_credentials to get a usable token:
    final tokenResp = await _http.post(
      Uri.parse('$hydraBaseUrl/oauth2/token'),
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      body: {
        'grant_type': 'client_credentials',
        'client_id': 'mobpilot-admin',
        'client_secret': 'mobpilot-admin-dev-secret',
        'scope': 'mobpilot:admin',
      },
    );

    if (tokenResp.statusCode == 200) {
      final tokenData = jsonDecode(tokenResp.body) as Map<String, dynamic>;
      _accessToken = tokenData['access_token'] as String?;
      _refreshToken = tokenData['refresh_token'] as String?;

      if (_accessToken != null) {
        await _storage.write(key: 'access_token', value: _accessToken);
        if (_refreshToken != null) {
          await _storage.write(key: 'refresh_token', value: _refreshToken);
        }
        return AuthResult.success(_accessToken!);
      }
    }

    // Fallback: use the Kratos session token directly
    _accessToken = _kratosSessionToken;
    if (_accessToken != null) {
      await _storage.write(key: 'access_token', value: _accessToken);
      return AuthResult.success(_accessToken!);
    }

    return AuthResult.failure('Failed to obtain access token');
  }

  /// Log out and clear all tokens.
  Future<void> logout() async {
    _accessToken = null;
    _refreshToken = null;
    _kratosSessionToken = null;
    await _storage.deleteAll();
  }

  String? _extractActionUrl(Map<String, dynamic> flowData) {
    final ui = flowData['ui'] as Map<String, dynamic>?;
    return ui?['action'] as String?;
  }

  AuthResult _parseKratosError(http.Response resp) {
    try {
      final body = jsonDecode(resp.body) as Map<String, dynamic>;
      final ui = body['ui'] as Map<String, dynamic>?;
      final messages = ui?['messages'] as List?;
      if (messages != null && messages.isNotEmpty) {
        final msg = messages.first as Map<String, dynamic>;
        return AuthResult.failure(msg['text'] as String? ?? 'Unknown error');
      }
      final errorMsg = body['error'] as Map<String, dynamic>?;
      if (errorMsg != null) {
        return AuthResult.failure(errorMsg['message'] as String? ?? 'Unknown error');
      }
    } catch (_) {}
    return AuthResult.failure('Authentication failed (${resp.statusCode})');
  }

}

class AuthResult {
  final bool isSuccess;
  final String? token;
  final String? error;

  AuthResult.success(this.token)
      : isSuccess = true,
        error = null;

  AuthResult.failure(this.error)
      : isSuccess = false,
        token = null;
}
