import 'package:flutter_test/flutter_test.dart';
import 'package:streaks/models/habit.dart';
import 'package:streaks/models/habit_log.dart';
import 'package:streaks/models/streak.dart';
import 'package:streaks/config/app_config.dart';

void main() {
  group('Habit model', () {
    test('creates with defaults', () {
      final habit = Habit(name: 'Test Habit');
      expect(habit.name, 'Test Habit');
      expect(habit.id, isNotEmpty);
      expect(habit.icon, HabitIcon.custom);
      expect(habit.frequency, HabitFrequency.daily);
      expect(habit.isArchived, false);
    });

    test('serializes to/from JSON', () {
      final habit = Habit(
        name: 'Run',
        icon: HabitIcon.exercise,
        color: '#FF6B6B',
        frequency: HabitFrequency.weekdays,
      );
      final json = habit.toJson();
      final restored = Habit.fromJson(json);
      expect(restored.name, 'Run');
      expect(restored.icon, HabitIcon.exercise);
      expect(restored.color, '#FF6B6B');
      expect(restored.frequency, HabitFrequency.weekdays);
    });

    test('copyWith preserves id', () {
      final habit = Habit(name: 'Original');
      final updated = habit.copyWith(name: 'Updated');
      expect(updated.id, habit.id);
      expect(updated.name, 'Updated');
    });
  });

  group('HabitLog', () {
    test('tracks completions', () {
      final log = HabitLog(year: 2026, month: 4);
      final date = DateTime(2026, 4, 5);

      expect(log.isCompleted('h1', date), false);
      log.toggle('h1', date);
      expect(log.isCompleted('h1', date), true);
      log.toggle('h1', date);
      expect(log.isCompleted('h1', date), false);
    });

    test('generates correct artifact key', () {
      final log = HabitLog(year: 2026, month: 4);
      expect(log.artifactKey, 'log/2026-04');
    });

    test('serializes to/from JSON', () {
      final log = HabitLog(year: 2026, month: 4);
      log.toggle('h1', DateTime(2026, 4, 1));
      log.toggle('h2', DateTime(2026, 4, 1));
      log.toggle('h1', DateTime(2026, 4, 2));

      final json = log.toJson();
      final restored = HabitLog.fromJson(2026, 4, json);
      expect(restored.isCompleted('h1', DateTime(2026, 4, 1)), true);
      expect(restored.isCompleted('h2', DateTime(2026, 4, 1)), true);
      expect(restored.isCompleted('h1', DateTime(2026, 4, 2)), true);
      expect(restored.isCompleted('h2', DateTime(2026, 4, 2)), false);
    });

    test('parses artifact key', () {
      final result = HabitLog.parseKey('log/2026-04');
      expect(result, (2026, 4));

      expect(HabitLog.parseKey('invalid'), null);
      expect(HabitLog.parseKey('log/bad'), null);
    });
  });

  group('StreakData', () {
    test('records consecutive completions', () {
      final data = StreakData();
      data.recordCompletion('h1', DateTime(2026, 4, 1));
      expect(data.getStreak('h1').current, 1);

      data.recordCompletion('h1', DateTime(2026, 4, 2));
      expect(data.getStreak('h1').current, 2);

      data.recordCompletion('h1', DateTime(2026, 4, 3));
      expect(data.getStreak('h1').current, 3);
      expect(data.getStreak('h1').best, 3);
    });

    test('resets streak on gap', () {
      final data = StreakData();
      data.recordCompletion('h1', DateTime(2026, 4, 1));
      data.recordCompletion('h1', DateTime(2026, 4, 2));
      expect(data.getStreak('h1').current, 2);

      // Skip a day
      data.recordCompletion('h1', DateTime(2026, 4, 4));
      expect(data.getStreak('h1').current, 1);
      expect(data.getStreak('h1').best, 2); // best preserved
    });

    test('serializes to/from JSON', () {
      final data = StreakData();
      data.recordCompletion('h1', DateTime(2026, 4, 1));
      data.recordCompletion('h1', DateTime(2026, 4, 2));

      final json = data.toJson();
      final restored = StreakData.fromJson(json);
      expect(restored.getStreak('h1').current, 2);
      expect(restored.getStreak('h1').best, 2);
    });
  });

  group('AppConfig', () {
    test('parses CLI args', () {
      final config = AppConfig.fromArgs([
        '--base-url', 'http://example.com/api',
        '--dev-token', 'test-token-123',
        '--test-mode',
      ]);
      expect(config.identityBaseUrl, 'http://example.com/api');
      expect(config.devToken, 'test-token-123');
      expect(config.testMode, true);
    });

    test('uses defaults when no args', () {
      final config = AppConfig.fromArgs([]);
      expect(config.identityBaseUrl, 'http://localhost/v1/identity');
      expect(config.devToken, null);
      expect(config.testMode, false);
    });
  });
}
