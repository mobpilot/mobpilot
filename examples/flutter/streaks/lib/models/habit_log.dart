import 'package:intl/intl.dart';

/// A month's worth of habit completion data.
///
/// Stored as a mobpilot artifact with key `log/YYYY-MM`.
/// Value shape: `{"2026-04-01": ["habit-id-1", "habit-id-3"], ...}`
class HabitLog {
  final int year;
  final int month;
  final Map<String, Set<String>> _entries;

  HabitLog({required this.year, required this.month, Map<String, Set<String>>? entries})
      : _entries = entries ?? {};

  /// Artifact key for this month's log.
  String get artifactKey => 'log/${year.toString().padLeft(4, '0')}-${month.toString().padLeft(2, '0')}';

  /// Get completed habit IDs for a given date.
  Set<String> completionsFor(DateTime date) {
    final key = _dateKey(date);
    return _entries[key] ?? {};
  }

  /// Check if a habit was completed on a date.
  bool isCompleted(String habitId, DateTime date) =>
      completionsFor(date).contains(habitId);

  /// Toggle a habit completion for a date. Returns true if now completed.
  bool toggle(String habitId, DateTime date) {
    final key = _dateKey(date);
    final set = _entries.putIfAbsent(key, () => {});
    if (set.contains(habitId)) {
      set.remove(habitId);
      return false;
    } else {
      set.add(habitId);
      return true;
    }
  }

  /// Total completions in this month.
  int get totalCompletions =>
      _entries.values.fold(0, (sum, set) => sum + set.length);

  /// Days with at least one completion.
  int get activeDays => _entries.values.where((s) => s.isNotEmpty).length;

  Map<String, dynamic> toJson() => _entries.map(
        (key, value) => MapEntry(key, value.toList()),
      );

  factory HabitLog.fromJson(int year, int month, Map<String, dynamic> json) {
    final entries = <String, Set<String>>{};
    for (final entry in json.entries) {
      final list = (entry.value as List).cast<String>();
      entries[entry.key] = list.toSet();
    }
    return HabitLog(year: year, month: month, entries: entries);
  }

  /// Parse artifact key like "log/2026-04" to extract year/month.
  static (int year, int month)? parseKey(String key) {
    final match = RegExp(r'^log/(\d{4})-(\d{2})$').firstMatch(key);
    if (match == null) return null;
    return (int.parse(match.group(1)!), int.parse(match.group(2)!));
  }

  static String _dateKey(DateTime date) =>
      DateFormat('yyyy-MM-dd').format(date);
}
