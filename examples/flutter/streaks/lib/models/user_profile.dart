/// Mobpilot user profile from the Identity API.
class UserProfile {
  final String id;
  final String appId;
  final String? displayName;
  final String? avatarUrl;
  final String? bio;
  final String? location;
  final String? websiteUrl;
  final Map<String, dynamic>? settings;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  const UserProfile({
    required this.id,
    required this.appId,
    this.displayName,
    this.avatarUrl,
    this.bio,
    this.location,
    this.websiteUrl,
    this.settings,
    this.createdAt,
    this.updatedAt,
  });

  factory UserProfile.fromJson(Map<String, dynamic> json) => UserProfile(
        id: json['id'] as String,
        appId: json['app_id'] as String,
        displayName: json['display_name'] as String?,
        avatarUrl: json['avatar_url'] as String?,
        bio: json['bio'] as String?,
        location: json['location'] as String?,
        websiteUrl: json['website_url'] as String?,
        settings: json['settings'] as Map<String, dynamic>?,
        createdAt: json['created_at'] != null
            ? DateTime.parse(json['created_at'] as String)
            : null,
        updatedAt: json['updated_at'] != null
            ? DateTime.parse(json['updated_at'] as String)
            : null,
      );
}
