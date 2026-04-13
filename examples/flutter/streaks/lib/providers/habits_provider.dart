import 'package:flutter/foundation.dart';
import '../models/habit.dart';
import '../models/habit_log.dart';
import '../models/streak.dart';
import '../services/mobpilot_client.dart';

/// Manages habits, logs, and streaks — backed by Mobpilot artifact storage.
class HabitsProvider extends ChangeNotifier {
  final MobpilotClient _client;

  List<Habit> _habits = [];
  final Map<String, HabitLog> _logs = {};
  StreakData _streaks = StreakData();
  bool _isLoading = false;
  String? _error;

  HabitsProvider(this._client);

  List<Habit> get habits => _habits.where((h) => !h.isArchived).toList();
  List<Habit> get archivedHabits => _habits.where((h) => h.isArchived).toList();
  StreakData get streaks => _streaks;
  bool get isLoading => _isLoading;
  String? get error => _error;

  /// Load all habit data from Mobpilot artifacts.
  Future<void> loadAll() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      // Load habits list
      final habitsArtifact = await _client.getArtifact('habits');
      if (habitsArtifact != null && habitsArtifact.value is List) {
        _habits = (habitsArtifact.value as List)
            .cast<Map<String, dynamic>>()
            .map(Habit.fromJson)
            .toList();
      }

      // Load streaks
      final streaksArtifact = await _client.getArtifact('streaks');
      if (streaksArtifact != null && streaksArtifact.value is Map) {
        _streaks = StreakData.fromJson(
            (streaksArtifact.value as Map).cast<String, dynamic>());
      }

      // Load current month's log
      final now = DateTime.now();
      await _loadMonthLog(now.year, now.month);

      // Also load previous month for streak continuity
      final prevMonth = DateTime(now.year, now.month - 1);
      await _loadMonthLog(prevMonth.year, prevMonth.month);
    } catch (e) {
      _error = 'Failed to load habits: $e';
    }

    _isLoading = false;
    notifyListeners();
  }

  /// Load a specific month's completion log.
  Future<void> _loadMonthLog(int year, int month) async {
    final log = HabitLog(year: year, month: month);
    final artifact = await _client.getArtifact(log.artifactKey);
    if (artifact != null && artifact.value is Map) {
      _logs[log.artifactKey] = HabitLog.fromJson(
          year, month, (artifact.value as Map).cast<String, dynamic>());
    } else {
      _logs[log.artifactKey] = log;
    }
  }

  /// Get the log for a given date's month.
  HabitLog getLogForDate(DateTime date) {
    final key =
        'log/${date.year.toString().padLeft(4, '0')}-${date.month.toString().padLeft(2, '0')}';
    return _logs[key] ?? HabitLog(year: date.year, month: date.month);
  }

  /// Check if a habit is completed for a given date.
  bool isCompleted(String habitId, DateTime date) =>
      getLogForDate(date).isCompleted(habitId, date);

  /// Add a new habit.
  Future<void> addHabit(Habit habit) async {
    _habits.add(habit);
    notifyListeners();
    await _saveHabits();
  }

  /// Update an existing habit.
  Future<void> updateHabit(Habit updated) async {
    final index = _habits.indexWhere((h) => h.id == updated.id);
    if (index >= 0) {
      _habits[index] = updated;
      notifyListeners();
      await _saveHabits();
    }
  }

  /// Archive a habit (soft delete).
  Future<void> archiveHabit(String habitId) async {
    final index = _habits.indexWhere((h) => h.id == habitId);
    if (index >= 0) {
      _habits[index] = _habits[index].copyWith(archivedAt: DateTime.now());
      notifyListeners();
      await _saveHabits();
    }
  }

  /// Toggle a habit completion for today.
  Future<void> toggleToday(String habitId) async {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    await toggleCompletion(habitId, today);
  }

  /// Toggle a habit completion for a specific date.
  Future<void> toggleCompletion(String habitId, DateTime date) async {
    // Ensure we have the log for this month loaded
    final logKey =
        'log/${date.year.toString().padLeft(4, '0')}-${date.month.toString().padLeft(2, '0')}';
    final log = _logs.putIfAbsent(
        logKey, () => HabitLog(year: date.year, month: date.month));

    final wasCompleted = log.isCompleted(habitId, date);
    log.toggle(habitId, date);

    // Update streaks
    if (!wasCompleted) {
      _streaks.recordCompletion(habitId, date);
    } else {
      _streaks.removeCompletion(habitId, date);
    }

    notifyListeners();

    // Persist to backend
    await Future.wait([
      _client.putArtifact(logKey, log.toJson()),
      _client.putArtifact('streaks', _streaks.toJson()),
    ]);
  }

  /// Get today's completion rate as a percentage.
  double get todayCompletionRate {
    final activeHabits = habits;
    if (activeHabits.isEmpty) return 0;

    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final completed = activeHabits.where((h) => isCompleted(h.id, today)).length;
    return completed / activeHabits.length;
  }

  /// Get the number of habits completed today.
  int get todayCompletedCount {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    return habits.where((h) => isCompleted(h.id, today)).length;
  }

  Future<void> _saveHabits() async {
    await _client.putArtifact(
      'habits',
      _habits.map((h) => h.toJson()).toList(),
    );
  }
}
