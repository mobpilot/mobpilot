import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/user_profile.dart';

/// HTTP client for the Mobpilot Identity API.
///
/// Wraps all REST calls to `/v1/identity/*` with Bearer token authentication.
class MobpilotClient {
  final String baseUrl;
  String? _token;
  final http.Client _http;

  MobpilotClient({required this.baseUrl, String? token, http.Client? httpClient})
      : _token = token,
        _http = httpClient ?? http.Client();

  String? get token => _token;

  void setToken(String token) => _token = token;

  void clearToken() => _token = null;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  // ── User Profile ──────────────────────────────────────────────────────

  /// Get current user profile (auto-creates on first call).
  Future<UserProfile> getMe() async {
    final resp = await _http.get(Uri.parse('$baseUrl/me'), headers: _headers);
    _checkResponse(resp);
    return UserProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Update current user profile.
  Future<UserProfile> updateMe({
    String? displayName,
    String? avatarUrl,
    String? bio,
    String? location,
    String? websiteUrl,
  }) async {
    final body = <String, dynamic>{};
    if (displayName != null) body['display_name'] = displayName;
    if (avatarUrl != null) body['avatar_url'] = avatarUrl;
    if (bio != null) body['bio'] = bio;
    if (location != null) body['location'] = location;
    if (websiteUrl != null) body['website_url'] = websiteUrl;

    final resp = await _http.patch(
      Uri.parse('$baseUrl/me'),
      headers: _headers,
      body: jsonEncode(body),
    );
    _checkResponse(resp);
    return UserProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Get a public user profile.
  Future<UserProfile> getUser(String userId) async {
    final resp = await _http.get(
      Uri.parse('$baseUrl/users/$userId'),
      headers: _headers,
    );
    _checkResponse(resp);
    return UserProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  // ── Device Tokens ─────────────────────────────────────────────────────

  /// Register a push notification device token.
  Future<String> registerDevice({
    required String platform,
    required String tokenType,
    required String token,
    required String deviceId,
  }) async {
    final resp = await _http.post(
      Uri.parse('$baseUrl/devices'),
      headers: _headers,
      body: jsonEncode({
        'platform': platform,
        'token_type': tokenType,
        'token': token,
        'device_id': deviceId,
      }),
    );
    _checkResponse(resp);
    final data = jsonDecode(resp.body) as Map<String, dynamic>;
    return data['id'] as String;
  }

  /// Unregister a device token.
  Future<void> unregisterDevice(String deviceId) async {
    final resp = await _http.delete(
      Uri.parse('$baseUrl/devices/$deviceId'),
      headers: _headers,
    );
    _checkResponse(resp);
  }

  // ── Artifact Storage ──────────────────────────────────────────────────

  /// List all artifacts for the current user.
  Future<List<ArtifactEntry>> listArtifacts() async {
    final resp = await _http.get(
      Uri.parse('$baseUrl/me/artifacts'),
      headers: _headers,
    );
    _checkResponse(resp);
    final data = jsonDecode(resp.body) as Map<String, dynamic>;
    final items = data['items'] as List? ?? [];
    return items
        .cast<Map<String, dynamic>>()
        .map(ArtifactEntry.fromJson)
        .toList();
  }

  /// Get an artifact by key.
  Future<ArtifactEntry?> getArtifact(String key) async {
    final resp = await _http.get(
      Uri.parse('$baseUrl/me/artifacts/$key'),
      headers: _headers,
    );
    if (resp.statusCode == 404) return null;
    _checkResponse(resp);
    return ArtifactEntry.fromJson(
        jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Create or replace an artifact.
  Future<ArtifactEntry> putArtifact(String key, dynamic value) async {
    final resp = await _http.put(
      Uri.parse('$baseUrl/me/artifacts/$key'),
      headers: _headers,
      body: jsonEncode(value),
    );
    _checkResponse(resp);
    return ArtifactEntry.fromJson(
        jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Delete an artifact.
  Future<void> deleteArtifact(String key) async {
    final resp = await _http.delete(
      Uri.parse('$baseUrl/me/artifacts/$key'),
      headers: _headers,
    );
    _checkResponse(resp);
  }

  // ── Helpers ───────────────────────────────────────────────────────────

  void _checkResponse(http.Response resp) {
    if (resp.statusCode >= 200 && resp.statusCode < 300) return;
    if (resp.statusCode == 401) {
      throw MobpilotAuthException('Authentication required');
    }
    throw MobpilotApiException(resp.statusCode, resp.body);
  }
}

/// A single artifact entry from the Identity API.
class ArtifactEntry {
  final String? id;
  final String key;
  final dynamic value;
  final DateTime? updatedAt;

  const ArtifactEntry({this.id, required this.key, this.value, this.updatedAt});

  factory ArtifactEntry.fromJson(Map<String, dynamic> json) => ArtifactEntry(
        id: json['id'] as String?,
        key: json['key'] as String,
        value: json['value'],
        updatedAt: json['updated_at'] != null
            ? DateTime.parse(json['updated_at'] as String)
            : null,
      );
}

class MobpilotAuthException implements Exception {
  final String message;
  MobpilotAuthException(this.message);
  @override
  String toString() => 'MobpilotAuthException: $message';
}

class MobpilotApiException implements Exception {
  final int statusCode;
  final String body;
  MobpilotApiException(this.statusCode, this.body);
  @override
  String toString() => 'MobpilotApiException($statusCode): $body';
}
