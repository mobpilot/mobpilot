import 'package:uuid/uuid.dart';

const _uuid = Uuid();

/// Icon/emoji identifiers for habits.
enum HabitIcon {
  exercise('exercise', '🏃'),
  water('water', '💧'),
  read('read', '📖'),
  meditate('meditate', '🧘'),
  sleep('sleep', '😴'),
  code('code', '💻'),
  eat('eat', '🥗'),
  journal('journal', '📝'),
  music('music', '🎵'),
  custom('custom', '⭐');

  const HabitIcon(this.id, this.emoji);
  final String id;
  final String emoji;

  static HabitIcon fromId(String id) =>
      HabitIcon.values.firstWhere((e) => e.id == id, orElse: () => HabitIcon.custom);
}

/// How often the habit should be performed.
enum HabitFrequency {
  daily,
  weekdays,
  weekends,
  threePerWeek,
  custom;

  String get displayName => switch (this) {
        daily => 'Every day',
        weekdays => 'Weekdays',
        weekends => 'Weekends',
        threePerWeek => '3x per week',
        custom => 'Custom',
      };
}

/// A single habit definition.
class Habit {
  final String id;
  final String name;
  final HabitIcon icon;
  final String color;
  final HabitFrequency frequency;
  final DateTime createdAt;
  final DateTime? archivedAt;

  Habit({
    String? id,
    required this.name,
    this.icon = HabitIcon.custom,
    this.color = '#6C63FF',
    this.frequency = HabitFrequency.daily,
    DateTime? createdAt,
    this.archivedAt,
  })  : id = id ?? _uuid.v4(),
        createdAt = createdAt ?? DateTime.now();

  bool get isArchived => archivedAt != null;

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'icon': icon.id,
        'color': color,
        'frequency': frequency.name,
        'created_at': createdAt.toIso8601String(),
        if (archivedAt != null) 'archived_at': archivedAt!.toIso8601String(),
      };

  factory Habit.fromJson(Map<String, dynamic> json) => Habit(
        id: json['id'] as String,
        name: json['name'] as String,
        icon: HabitIcon.fromId(json['icon'] as String? ?? 'custom'),
        color: json['color'] as String? ?? '#6C63FF',
        frequency: HabitFrequency.values.firstWhere(
          (f) => f.name == json['frequency'],
          orElse: () => HabitFrequency.daily,
        ),
        createdAt: DateTime.parse(json['created_at'] as String),
        archivedAt: json['archived_at'] != null
            ? DateTime.parse(json['archived_at'] as String)
            : null,
      );

  Habit copyWith({
    String? name,
    HabitIcon? icon,
    String? color,
    HabitFrequency? frequency,
    DateTime? archivedAt,
  }) =>
      Habit(
        id: id,
        name: name ?? this.name,
        icon: icon ?? this.icon,
        color: color ?? this.color,
        frequency: frequency ?? this.frequency,
        createdAt: createdAt,
        archivedAt: archivedAt ?? this.archivedAt,
      );
}
