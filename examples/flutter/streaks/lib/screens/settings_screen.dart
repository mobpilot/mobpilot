import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../config/app_config.dart';
import '../providers/auth_provider.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final config = context.read<AppConfig>();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          // Account section
          _sectionHeader(theme, 'Account'),
          ListTile(
            leading: const Icon(Icons.logout),
            title: const Text('Sign Out'),
            onTap: () async {
              final confirmed = await showDialog<bool>(
                context: context,
                builder: (ctx) => AlertDialog(
                  title: const Text('Sign Out'),
                  content:
                      const Text('Are you sure you want to sign out?'),
                  actions: [
                    TextButton(
                      onPressed: () => Navigator.pop(ctx, false),
                      child: const Text('Cancel'),
                    ),
                    FilledButton(
                      onPressed: () => Navigator.pop(ctx, true),
                      child: const Text('Sign Out'),
                    ),
                  ],
                ),
              );
              if (confirmed == true && context.mounted) {
                await context.read<AuthProvider>().logout();
              }
            },
          ),

          const Divider(),

          // About section
          _sectionHeader(theme, 'About'),
          ListTile(
            leading: const Icon(Icons.info_outline),
            title: const Text('About Streaks'),
            subtitle: const Text('Reference app for Mobpilot'),
            onTap: () => _showAbout(context),
          ),

          // Debug info (visible when test mode or dev token)
          if (config.testMode || config.devToken != null) ...[
            const Divider(),
            _sectionHeader(theme, 'Debug'),
            ListTile(
              leading: const Icon(Icons.api),
              title: const Text('API Base URL'),
              subtitle: Text(config.identityBaseUrl),
            ),
            ListTile(
              leading: const Icon(Icons.vpn_key),
              title: const Text('Auth Mode'),
              subtitle: Text(
                config.devToken != null ? 'Dev Token' : 'Ory Kratos/Hydra',
              ),
            ),
            if (config.testMode)
              ListTile(
                leading: const Icon(Icons.bug_report),
                title: const Text('Test Mode'),
                subtitle: const Text('Enabled'),
              ),
          ],
        ],
      ),
    );
  }

  Widget _sectionHeader(ThemeData theme, String title) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
      child: Text(
        title,
        style: theme.textTheme.titleSmall?.copyWith(
          color: theme.colorScheme.primary,
          fontWeight: FontWeight.bold,
        ),
      ),
    );
  }

  void _showAbout(BuildContext context) {
    showAboutDialog(
      context: context,
      applicationName: 'Streaks',
      applicationVersion: '1.0.0',
      applicationIcon: const Icon(
        Icons.local_fire_department_rounded,
        size: 48,
        color: Colors.deepPurple,
      ),
      children: [
        const Text(
          'Streaks is a habit tracker with social accountability, '
          'built as a reference implementation for the Mobpilot '
          'mobile backend platform.\n\n'
          'It demonstrates:\n'
          '- User authentication via Ory Kratos/Hydra\n'
          '- Profile management via Identity API\n'
          '- Data storage via Artifact API\n'
          '- Push notification token registration\n\n'
          'https://github.com/mobpilot/mobpilot',
        ),
      ],
    );
  }
}
