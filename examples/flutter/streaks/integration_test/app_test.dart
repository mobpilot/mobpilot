import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:streaks/app.dart';
import 'package:streaks/config/app_config.dart';

/// Integration tests for the Streaks app.
///
/// Run against a live Mobpilot instance:
///   flutter test integration_test/app_test.dart \
///     --dart-define=BASE_URL=http://localhost/v1/identity \
///     --dart-define=DEV_TOKEN=<token>
///
/// Or from the command line with arguments:
///   flutter test integration_test/app_test.dart
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // Read config from environment (set via --dart-define)
  const baseUrl = String.fromEnvironment(
    'BASE_URL',
    defaultValue: 'http://localhost/v1/identity',
  );
  const devToken = String.fromEnvironment('DEV_TOKEN');

  final config = AppConfig(
    identityBaseUrl: baseUrl,
    devToken: devToken.isNotEmpty ? devToken : null,
    testMode: true,
  );

  group('App launch', () {
    testWidgets('shows login screen when not authenticated', (tester) async {
      final testConfig = AppConfig(
        identityBaseUrl: baseUrl,
        testMode: true,
        // No dev token → should show login
      );
      await tester.pumpWidget(StreaksApp(config: testConfig));
      await tester.pumpAndSettle();

      expect(find.text('Streaks'), findsOneWidget);
      expect(find.text('Sign In'), findsOneWidget);
      expect(find.text("Don't have an account? Sign up"), findsOneWidget);
    });

    testWidgets('shows home screen when dev token is provided', (tester) async {
      if (devToken.isEmpty) {
        // Skip if no dev token is available
        return;
      }

      await tester.pumpWidget(StreaksApp(config: config));
      await tester.pumpAndSettle(const Duration(seconds: 3));

      // Should show home screen with empty state or habits
      expect(find.byType(Scaffold), findsWidgets);
    });
  });

  group('Login screen', () {
    testWidgets('validates empty fields', (tester) async {
      final testConfig = AppConfig(
        identityBaseUrl: baseUrl,
        testMode: true,
      );
      await tester.pumpWidget(StreaksApp(config: testConfig));
      await tester.pumpAndSettle();

      // Tap sign in without filling fields
      await tester.tap(find.text('Sign In'));
      await tester.pumpAndSettle();

      expect(find.text('Email is required'), findsOneWidget);
      expect(find.text('Password is required'), findsOneWidget);
    });

    testWidgets('navigates to register screen', (tester) async {
      final testConfig = AppConfig(
        identityBaseUrl: baseUrl,
        testMode: true,
      );
      await tester.pumpWidget(StreaksApp(config: testConfig));
      await tester.pumpAndSettle();

      await tester.tap(find.text("Don't have an account? Sign up"));
      await tester.pumpAndSettle();

      expect(find.text('Join Streaks'), findsOneWidget);
      expect(find.text('Create Account'), findsOneWidget);
    });
  });

  group('Habit management (with dev token)', () {
    testWidgets('can add a new habit', (tester) async {
      if (devToken.isEmpty) return;

      await tester.pumpWidget(StreaksApp(config: config));
      await tester.pumpAndSettle(const Duration(seconds: 3));

      // Tap FAB to add habit
      await tester.tap(find.text('New Habit'));
      await tester.pumpAndSettle();

      // Fill in habit name
      await tester.enterText(
        find.byType(TextFormField).first,
        'Integration Test Habit',
      );

      // Save
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle(const Duration(seconds: 2));

      // Should be back on home screen with the habit visible
      expect(find.text('Integration Test Habit'), findsOneWidget);
    });

    testWidgets('can toggle habit completion', (tester) async {
      if (devToken.isEmpty) return;

      await tester.pumpWidget(StreaksApp(config: config));
      await tester.pumpAndSettle(const Duration(seconds: 3));

      // Find the first habit card's checkbox (circle)
      final checkboxes = find.byIcon(Icons.circle_outlined);
      if (checkboxes.evaluate().isEmpty) return; // No habits

      // Toggle it (the GestureDetector wrapping the checkbox)
      // We need to find the actual toggle area in the habit card
      final habitCards = find.byType(Card);
      if (habitCards.evaluate().isEmpty) return;

      // Tap the first card to open detail
      await tester.tap(habitCards.first);
      await tester.pumpAndSettle();

      // Look for "Mark as Done" button
      final markDone = find.text('Mark as Done');
      if (markDone.evaluate().isNotEmpty) {
        await tester.tap(markDone);
        await tester.pumpAndSettle(const Duration(seconds: 2));

        // Should now show "Completed Today!"
        expect(find.text('Completed Today!'), findsOneWidget);
      }
    });
  });

  group('Profile', () {
    testWidgets('can navigate to profile screen', (tester) async {
      if (devToken.isEmpty) return;

      await tester.pumpWidget(StreaksApp(config: config));
      await tester.pumpAndSettle(const Duration(seconds: 3));

      // Tap profile icon in app bar
      await tester.tap(find.byIcon(Icons.person_outlined));
      await tester.pumpAndSettle();

      expect(find.text('Profile'), findsOneWidget);
      expect(find.text('Stats'), findsOneWidget);
    });
  });

  group('Settings', () {
    testWidgets('can navigate to settings and see debug info', (tester) async {
      if (devToken.isEmpty) return;

      await tester.pumpWidget(StreaksApp(config: config));
      await tester.pumpAndSettle(const Duration(seconds: 3));

      // Tap settings icon
      await tester.tap(find.byIcon(Icons.settings_outlined));
      await tester.pumpAndSettle();

      expect(find.text('Settings'), findsOneWidget);
      expect(find.text('Debug'), findsOneWidget);
      expect(find.text('Dev Token'), findsOneWidget);
    });
  });
}
