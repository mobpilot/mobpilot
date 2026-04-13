/// Streak data for all habits.
///
/// Stored as a mobpilot artifact with key `streaks`.
/// Value shape: `{"habit-id": {"current": 14, "best": 30, "last_date": "2026-04-05"}}`
class StreakData {
  final Map<String, HabitStreak> _streaks;

  StreakData([Map<String, HabitStreak>? streaks])
      : _streaks = streaks ?? {};

  HabitStreak getStreak(String habitId) =>
      _streaks[habitId] ?? const HabitStreak();

  /// Update streak after a check-in.
  void recordCompletion(String habitId, DateTime date) {
    final current = getStreak(habitId);
    final dateStr = _dateStr(date);
    final yesterday = _dateStr(date.subtract(const Duration(days: 1)));

    int newCurrent;
    if (current.lastDate == yesterday) {
      newCurrent = current.current + 1;
    } else if (current.lastDate == dateStr) {
      newCurrent = current.current; // already counted
    } else {
      newCurrent = 1; // streak broken, start fresh
    }

    final newBest = newCurrent > current.best ? newCurrent : current.best;
    _streaks[habitId] = HabitStreak(
      current: newCurrent,
      best: newBest,
      lastDate: dateStr,
    );
  }

  /// Undo a completion — recompute streak from scratch would be better,
  /// but for simplicity we just decrement if it was the last date.
  void removeCompletion(String habitId, DateTime date) {
    final current = getStreak(habitId);
    final dateStr = _dateStr(date);
    if (current.lastDate == dateStr && current.current > 0) {
      _streaks[habitId] = HabitStreak(
        current: current.current - 1,
        best: current.best,
        lastDate: _dateStr(date.subtract(const Duration(days: 1))),
      );
    }
  }

  Map<String, dynamic> toJson() =>
      _streaks.map((k, v) => MapEntry(k, v.toJson()));

  factory StreakData.fromJson(Map<String, dynamic> json) {
    final streaks = <String, HabitStreak>{};
    for (final entry in json.entries) {
      streaks[entry.key] =
          HabitStreak.fromJson(entry.value as Map<String, dynamic>);
    }
    return StreakData(streaks);
  }

  static String _dateStr(DateTime d) =>
      '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';
}

/// Streak info for a single habit.
class HabitStreak {
  final int current;
  final int best;
  final String? lastDate;

  const HabitStreak({this.current = 0, this.best = 0, this.lastDate});

  Map<String, dynamic> toJson() => {
        'current': current,
        'best': best,
        if (lastDate != null) 'last_date': lastDate,
      };

  factory HabitStreak.fromJson(Map<String, dynamic> json) => HabitStreak(
        current: json['current'] as int? ?? 0,
        best: json['best'] as int? ?? 0,
        lastDate: json['last_date'] as String?,
      );
}
