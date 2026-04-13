import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/habits_provider.dart';
import '../providers/profile_provider.dart';

class ProfileScreen extends StatefulWidget {
  const ProfileScreen({super.key});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  final _nameController = TextEditingController();
  final _bioController = TextEditingController();
  final _locationController = TextEditingController();
  bool _isEditing = false;

  @override
  void initState() {
    super.initState();
    final profile = context.read<ProfileProvider>().profile;
    if (profile != null) {
      _nameController.text = profile.displayName ?? '';
      _bioController.text = profile.bio ?? '';
      _locationController.text = profile.location ?? '';
    }
  }

  @override
  void dispose() {
    _nameController.dispose();
    _bioController.dispose();
    _locationController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final profileProvider = context.read<ProfileProvider>();
    final success = await profileProvider.update(
      displayName: _nameController.text.trim(),
      bio: _bioController.text.trim(),
      location: _locationController.text.trim(),
    );
    if (success) {
      setState(() => _isEditing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final profileProvider = context.watch<ProfileProvider>();
    final habits = context.watch<HabitsProvider>();
    final profile = profileProvider.profile;
    final theme = Theme.of(context);

    // Compute stats
    final activeCount = habits.habits.length;
    int bestStreak = 0;
    for (final h in habits.habits) {
      final s = habits.streaks.getStreak(h.id);
      if (s.best > bestStreak) bestStreak = s.best;
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('Profile'),
        actions: [
          if (!_isEditing)
            IconButton(
              icon: const Icon(Icons.edit_outlined),
              onPressed: () => setState(() => _isEditing = true),
            )
          else ...[
            TextButton(
              onPressed: () => setState(() => _isEditing = false),
              child: const Text('Cancel'),
            ),
            TextButton(
              onPressed: profileProvider.isLoading ? null : _save,
              child: const Text('Save'),
            ),
          ],
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Avatar + name
          Center(
            child: Column(
              children: [
                CircleAvatar(
                  radius: 48,
                  backgroundImage: profile?.avatarUrl != null
                      ? NetworkImage(profile!.avatarUrl!)
                      : null,
                  child: profile?.avatarUrl == null
                      ? Text(
                          (profile?.displayName ?? '?')
                              .substring(0, 1)
                              .toUpperCase(),
                          style: theme.textTheme.headlineMedium,
                        )
                      : null,
                ),
                const SizedBox(height: 12),
                if (!_isEditing) ...[
                  Text(
                    profile?.displayName ?? 'Set your name',
                    style: theme.textTheme.headlineSmall?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  if (profile?.bio != null && profile!.bio!.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      profile.bio!,
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      textAlign: TextAlign.center,
                    ),
                  ],
                  if (profile?.location != null &&
                      profile!.location!.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(Icons.location_on_outlined,
                            size: 16,
                            color: theme.colorScheme.onSurfaceVariant),
                        const SizedBox(width: 4),
                        Text(
                          profile.location!,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ],
                ],
              ],
            ),
          ),

          if (_isEditing) ...[
            const SizedBox(height: 24),
            TextFormField(
              controller: _nameController,
              decoration: const InputDecoration(
                labelText: 'Display Name',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _bioController,
              maxLines: 3,
              maxLength: 500,
              decoration: const InputDecoration(
                labelText: 'Bio',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _locationController,
              decoration: const InputDecoration(
                labelText: 'Location',
                border: OutlineInputBorder(),
              ),
            ),
          ],

          const SizedBox(height: 32),

          // Stats cards
          Text(
            'Stats',
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              _buildStatCard(
                  theme, 'Active Habits', '$activeCount', Icons.track_changes),
              const SizedBox(width: 12),
              _buildStatCard(
                  theme, 'Best Streak', '$bestStreak days', Icons.local_fire_department),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              _buildStatCard(
                theme,
                'Today',
                '${habits.todayCompletedCount}/$activeCount',
                Icons.today,
              ),
              const SizedBox(width: 12),
              _buildStatCard(
                theme,
                'Completion',
                '${(habits.todayCompletionRate * 100).round()}%',
                Icons.pie_chart_outline,
              ),
            ],
          ),

          if (profile?.createdAt != null) ...[
            const SizedBox(height: 24),
            Text(
              'Member since ${profile!.createdAt!.day}/${profile.createdAt!.month}/${profile.createdAt!.year}',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildStatCard(
      ThemeData theme, String label, String value, IconData icon) {
    return Expanded(
      child: Card(
        elevation: 0,
        color: theme.colorScheme.surfaceContainerHighest,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              Icon(icon, color: theme.colorScheme.primary),
              const SizedBox(height: 8),
              Text(
                value,
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                label,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
