import 'package:flutter/material.dart';
import '../models/habit.dart';
import '../models/streak.dart';

class HabitCard extends StatelessWidget {
  final Habit habit;
  final bool isCompleted;
  final HabitStreak streak;
  final VoidCallback onToggle;
  final VoidCallback onTap;

  const HabitCard({
    super.key,
    required this.habit,
    required this.isCompleted,
    required this.streak,
    required this.onToggle,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final habitColor = _hexToColor(habit.color);

    return Card(
      elevation: 0,
      color: isCompleted
          ? habitColor.withValues(alpha: 0.12)
          : theme.colorScheme.surfaceContainerLow,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: isCompleted
            ? BorderSide(color: habitColor.withValues(alpha: 0.3))
            : BorderSide.none,
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            children: [
              // Checkbox
              GestureDetector(
                onTap: onToggle,
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 200),
                  width: 32,
                  height: 32,
                  decoration: BoxDecoration(
                    color: isCompleted ? habitColor : Colors.transparent,
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: isCompleted
                          ? habitColor
                          : theme.colorScheme.outline,
                      width: 2,
                    ),
                  ),
                  child: isCompleted
                      ? const Icon(Icons.check, color: Colors.white, size: 18)
                      : null,
                ),
              ),
              const SizedBox(width: 12),

              // Icon + name
              Text(habit.icon.emoji, style: const TextStyle(fontSize: 24)),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      habit.name,
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                        decoration: isCompleted
                            ? TextDecoration.lineThrough
                            : null,
                        color: isCompleted
                            ? theme.colorScheme.onSurface.withValues(alpha: 0.6)
                            : null,
                      ),
                    ),
                    Text(
                      habit.frequency.displayName,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),

              // Streak badge
              if (streak.current > 0)
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: habitColor.withValues(alpha: 0.15),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.local_fire_department,
                        size: 16,
                        color: habitColor,
                      ),
                      const SizedBox(width: 4),
                      Text(
                        '${streak.current}',
                        style: theme.textTheme.labelMedium?.copyWith(
                          color: habitColor,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ],
                  ),
                ),

              const SizedBox(width: 8),
              Icon(
                Icons.chevron_right,
                color: theme.colorScheme.outline,
              ),
            ],
          ),
        ),
      ),
    );
  }

  static Color _hexToColor(String hex) {
    final value = int.parse(hex.replaceFirst('#', ''), radix: 16);
    return Color(value | 0xFF000000);
  }
}
