import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:table_calendar/table_calendar.dart';
import '../models/habit.dart';
import '../providers/habits_provider.dart';
import '../widgets/streak_counter.dart';
import 'add_habit_screen.dart';

class HabitDetailScreen extends StatelessWidget {
  final Habit habit;

  const HabitDetailScreen({super.key, required this.habit});

  @override
  Widget build(BuildContext context) {
    final habits = context.watch<HabitsProvider>();
    final streak = habits.streaks.getStreak(habit.id);
    final theme = Theme.of(context);
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final habitColor = _hexToColor(habit.color);

    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            Text(habit.icon.emoji, style: const TextStyle(fontSize: 24)),
            const SizedBox(width: 8),
            Text(habit.name),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit_outlined),
            onPressed: () => Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => AddHabitScreen(editHabit: habit),
              ),
            ),
          ),
          PopupMenuButton<String>(
            onSelected: (value) async {
              if (value == 'archive') {
                await habits.archiveHabit(habit.id);
                if (context.mounted) Navigator.pop(context);
              }
            },
            itemBuilder: (_) => [
              const PopupMenuItem(
                value: 'archive',
                child: Row(
                  children: [
                    Icon(Icons.archive_outlined),
                    SizedBox(width: 8),
                    Text('Archive'),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Streak cards
          Row(
            children: [
              Expanded(
                child: StreakCounter(
                  label: 'Current Streak',
                  count: streak.current,
                  color: habitColor,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: StreakCounter(
                  label: 'Best Streak',
                  count: streak.best,
                  color: theme.colorScheme.tertiary,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          _buildInfoRow(theme, 'Frequency', habit.frequency.displayName),
          _buildInfoRow(
            theme,
            'Started',
            '${habit.createdAt.day}/${habit.createdAt.month}/${habit.createdAt.year}',
          ),
          const SizedBox(height: 24),

          // Calendar
          Text(
            'History',
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          Card(
            elevation: 0,
            color: theme.colorScheme.surfaceContainerLow,
            child: Padding(
              padding: const EdgeInsets.all(8),
              child: TableCalendar(
                firstDay: habit.createdAt,
                lastDay: today,
                focusedDay: today,
                calendarFormat: CalendarFormat.month,
                startingDayOfWeek: StartingDayOfWeek.monday,
                headerStyle: HeaderStyle(
                  formatButtonVisible: false,
                  titleCentered: true,
                  titleTextStyle: theme.textTheme.titleSmall!,
                ),
                calendarStyle: CalendarStyle(
                  outsideDaysVisible: false,
                  todayDecoration: BoxDecoration(
                    border: Border.all(color: habitColor, width: 2),
                    shape: BoxShape.circle,
                  ),
                  todayTextStyle: TextStyle(color: theme.colorScheme.onSurface),
                ),
                calendarBuilders: CalendarBuilders(
                  defaultBuilder: (context, date, _) =>
                      _buildCalendarDay(context, date, habits, habitColor, false),
                  todayBuilder: (context, date, _) =>
                      _buildCalendarDay(context, date, habits, habitColor, true),
                ),
                onDaySelected: (selectedDay, _) {
                  if (!selectedDay.isAfter(today)) {
                    habits.toggleCompletion(habit.id, selectedDay);
                  }
                },
              ),
            ),
          ),

          const SizedBox(height: 24),

          // Today's action button
          SizedBox(
            height: 56,
            child: FilledButton.icon(
              onPressed: () => habits.toggleToday(habit.id),
              icon: Icon(
                habits.isCompleted(habit.id, today)
                    ? Icons.check_circle
                    : Icons.circle_outlined,
              ),
              label: Text(
                habits.isCompleted(habit.id, today)
                    ? 'Completed Today!'
                    : 'Mark as Done',
                style: const TextStyle(fontSize: 16),
              ),
              style: FilledButton.styleFrom(
                backgroundColor: habits.isCompleted(habit.id, today)
                    ? habitColor
                    : null,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCalendarDay(
    BuildContext context,
    DateTime date,
    HabitsProvider habits,
    Color habitColor,
    bool isToday,
  ) {
    final completed = habits.isCompleted(habit.id, date);

    return Container(
      margin: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: completed ? habitColor : null,
        shape: BoxShape.circle,
        border: isToday ? Border.all(color: habitColor, width: 2) : null,
      ),
      child: Center(
        child: Text(
          '${date.day}',
          style: TextStyle(
            color: completed
                ? Colors.white
                : Theme.of(context).colorScheme.onSurface,
            fontWeight: completed ? FontWeight.bold : FontWeight.normal,
          ),
        ),
      ),
    );
  }

  Widget _buildInfoRow(ThemeData theme, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Text(
            label,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const Spacer(),
          Text(value, style: theme.textTheme.bodyMedium),
        ],
      ),
    );
  }

  static Color _hexToColor(String hex) {
    final value = int.parse(hex.replaceFirst('#', ''), radix: 16);
    return Color(value | 0xFF000000);
  }
}
