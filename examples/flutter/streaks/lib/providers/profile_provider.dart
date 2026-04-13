import 'package:flutter/foundation.dart';
import '../models/user_profile.dart';
import '../services/mobpilot_client.dart';

/// Manages the current user's profile from the Mobpilot Identity API.
class ProfileProvider extends ChangeNotifier {
  final MobpilotClient _client;

  UserProfile? _profile;
  bool _isLoading = false;
  String? _error;

  ProfileProvider(this._client);

  UserProfile? get profile => _profile;
  bool get isLoading => _isLoading;
  String? get error => _error;

  /// Load the current user's profile (auto-creates on first call).
  Future<void> load() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _profile = await _client.getMe();
    } catch (e) {
      _error = 'Failed to load profile: $e';
    }

    _isLoading = false;
    notifyListeners();
  }

  /// Update the current user's profile.
  Future<bool> update({
    String? displayName,
    String? bio,
    String? location,
    String? avatarUrl,
  }) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _profile = await _client.updateMe(
        displayName: displayName,
        bio: bio,
        location: location,
        avatarUrl: avatarUrl,
      );
      _isLoading = false;
      notifyListeners();
      return true;
    } catch (e) {
      _error = 'Failed to update profile: $e';
      _isLoading = false;
      notifyListeners();
      return false;
    }
  }
}
