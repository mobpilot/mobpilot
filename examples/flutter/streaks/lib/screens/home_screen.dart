import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/habits_provider.dart';
import '../providers/profile_provider.dart';
import '../widgets/habit_card.dart';
import '../widgets/progress_ring.dart';
import 'add_habit_screen.dart';
import 'habit_detail_screen.dart';
import 'profile_screen.dart';
import 'settings_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  @override
  void initState() {
    super.initState();
    // Load data on first build
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<HabitsProvider>().loadAll();
      context.read<ProfileProvider>().load();
    });
  }

  @override
  Widget build(BuildContext context) {
    final habits = context.watch<HabitsProvider>();
    final profile = context.watch<ProfileProvider>();
    final theme = Theme.of(context);
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);

    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              _greeting(),
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            Text(
              profile.profile?.displayName ?? 'Streaks',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_outlined),
            onPressed: () => Navigator.push(
              context,
              MaterialPageRoute(builder: (_) => const ProfileScreen()),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.settings_outlined),
            onPressed: () => Navigator.push(
              context,
              MaterialPageRoute(builder: (_) => const SettingsScreen()),
            ),
          ),
        ],
      ),
      body: habits.isLoading
          ? const Center(child: CircularProgressIndicator())
          : habits.error != null
              ? _buildError(habits.error!)
              : _buildContent(context, habits, today, theme),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          await Navigator.push(
            context,
            MaterialPageRoute(builder: (_) => const AddHabitScreen()),
          );
        },
        icon: const Icon(Icons.add),
        label: const Text('New Habit'),
      ),
    );
  }

  Widget _buildError(String error) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.error_outline, size: 48, color: Colors.red),
          const SizedBox(height: 16),
          Text(error, textAlign: TextAlign.center),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: () => context.read<HabitsProvider>().loadAll(),
            child: const Text('Retry'),
          ),
        ],
      ),
    );
  }

  Widget _buildContent(
    BuildContext context,
    HabitsProvider habits,
    DateTime today,
    ThemeData theme,
  ) {
    final activeHabits = habits.habits;

    if (activeHabits.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.track_changes,
              size: 80,
              color: theme.colorScheme.primary.withValues(alpha: 0.3),
            ),
            const SizedBox(height: 24),
            Text(
              'No habits yet',
              style: theme.textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            Text(
              'Tap the button below to create your first habit',
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () => habits.loadAll(),
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Today's progress summary
          _buildProgressCard(habits, today, theme),
          const SizedBox(height: 24),

          // Section header
          Row(
            children: [
              Text(
                "Today's Habits",
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const Spacer(),
              Text(
                '${habits.todayCompletedCount}/${activeHabits.length}',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),

          // Habit cards
          ...activeHabits.map((habit) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: HabitCard(
                  habit: habit,
                  isCompleted: habits.isCompleted(habit.id, today),
                  streak: habits.streaks.getStreak(habit.id),
                  onToggle: () => habits.toggleToday(habit.id),
                  onTap: () => Navigator.push(
                    context,
                    MaterialPageRoute(
                      builder: (_) => HabitDetailScreen(habit: habit),
                    ),
                  ),
                ),
              )),

          const SizedBox(height: 80), // Space for FAB
        ],
      ),
    );
  }

  Widget _buildProgressCard(
      HabitsProvider habits, DateTime today, ThemeData theme) {
    final rate = habits.todayCompletionRate;

    return Card(
      elevation: 0,
      color: theme.colorScheme.primaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Row(
          children: [
            ProgressRing(
              progress: rate,
              size: 72,
              strokeWidth: 6,
              color: theme.colorScheme.primary,
              backgroundColor:
                  theme.colorScheme.primary.withValues(alpha: 0.2),
              child: Text(
                '${(rate * 100).round()}%',
                style: theme.textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: theme.colorScheme.onPrimaryContainer,
                ),
              ),
            ),
            const SizedBox(width: 20),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    rate >= 1.0
                        ? 'All done!'
                        : rate > 0.5
                            ? 'Keep going!'
                            : rate > 0
                                ? 'Good start!'
                                : 'Ready to begin?',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.bold,
                      color: theme.colorScheme.onPrimaryContainer,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${habits.todayCompletedCount} of ${habits.habits.length} habits completed today',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onPrimaryContainer
                          .withValues(alpha: 0.8),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _greeting() {
    final hour = DateTime.now().hour;
    if (hour < 12) return 'Good morning';
    if (hour < 17) return 'Good afternoon';
    return 'Good evening';
  }
}
