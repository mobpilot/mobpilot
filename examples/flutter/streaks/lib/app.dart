import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'config/app_config.dart';
import 'providers/auth_provider.dart';
import 'providers/habits_provider.dart';
import 'providers/profile_provider.dart';
import 'services/auth_service.dart';
import 'services/mobpilot_client.dart';
import 'screens/home_screen.dart';
import 'screens/login_screen.dart';

class StreaksApp extends StatelessWidget {
  final AppConfig config;

  const StreaksApp({super.key, required this.config});

  @override
  Widget build(BuildContext context) {
    final client = MobpilotClient(
      baseUrl: config.identityBaseUrl,
      token: config.devToken,
    );

    final authService = AuthService(
      kratosBaseUrl: config.kratosBaseUrl,
      hydraBaseUrl: config.hydraBaseUrl,
      clientId: config.oauthClientId,
      redirectUri: config.oauthRedirectUri,
    );

    return MultiProvider(
      providers: [
        Provider<AppConfig>.value(value: config),
        Provider<MobpilotClient>.value(value: client),
        ChangeNotifierProvider(
          create: (_) {
            final auth = AuthProvider(auth: authService, client: client);
            if (config.devToken != null) {
              auth.setDevToken(config.devToken!);
            } else {
              auth.init();
            }
            return auth;
          },
        ),
        ChangeNotifierProvider(create: (_) => HabitsProvider(client)),
        ChangeNotifierProvider(create: (_) => ProfileProvider(client)),
      ],
      child: MaterialApp(
        title: 'Streaks',
        debugShowCheckedModeBanner: false,
        theme: _buildTheme(Brightness.light),
        darkTheme: _buildTheme(Brightness.dark),
        home: const _AuthGate(),
      ),
    );
  }

  static ThemeData _buildTheme(Brightness brightness) {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: const Color(0xFF6C63FF),
      brightness: brightness,
    );
    return ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      cardTheme: CardThemeData(
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
    );
  }
}

/// Routes to login or home based on auth state.
class _AuthGate extends StatelessWidget {
  const _AuthGate();

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();

    if (auth.isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (auth.isAuthenticated) {
      return const HomeScreen();
    }

    return const LoginScreen();
  }
}
