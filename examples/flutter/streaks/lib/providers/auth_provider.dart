import 'package:flutter/foundation.dart';
import '../services/auth_service.dart';
import '../services/mobpilot_client.dart';

/// Manages authentication state and exposes the API client.
class AuthProvider extends ChangeNotifier {
  final AuthService _auth;
  final MobpilotClient client;

  bool _isLoading = false;
  bool _isAuthenticated = false;
  String? _error;

  AuthProvider({required AuthService auth, required this.client}) : _auth = auth;

  bool get isLoading => _isLoading;
  bool get isAuthenticated => _isAuthenticated;
  String? get error => _error;

  /// Try to restore a previous session on app start.
  Future<void> init() async {
    _isLoading = true;
    notifyListeners();

    final restored = await _auth.tryRestoreSession();
    if (restored && _auth.accessToken != null) {
      client.setToken(_auth.accessToken!);
      _isAuthenticated = true;
    }

    _isLoading = false;
    notifyListeners();
  }

  /// Set a dev token directly.
  Future<void> setDevToken(String token) async {
    await _auth.setDevToken(token);
    client.setToken(token);
    _isAuthenticated = true;
    notifyListeners();
  }

  /// Register a new account.
  Future<bool> register(String email, String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    final result = await _auth.register(email: email, password: password);

    if (result.isSuccess && result.token != null) {
      client.setToken(result.token!);
      _isAuthenticated = true;
      _isLoading = false;
      notifyListeners();
      return true;
    }

    _error = result.error;
    _isLoading = false;
    notifyListeners();
    return false;
  }

  /// Log in with existing credentials.
  Future<bool> login(String email, String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    final result = await _auth.login(email: email, password: password);

    if (result.isSuccess && result.token != null) {
      client.setToken(result.token!);
      _isAuthenticated = true;
      _isLoading = false;
      notifyListeners();
      return true;
    }

    _error = result.error;
    _isLoading = false;
    notifyListeners();
    return false;
  }

  /// Log out.
  Future<void> logout() async {
    await _auth.logout();
    client.clearToken();
    _isAuthenticated = false;
    _error = null;
    notifyListeners();
  }
}
